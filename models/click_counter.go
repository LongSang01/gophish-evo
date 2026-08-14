package models

import (
	"sync"
	"time"

	log "github.com/gophish/gophish/logger"
)

// PageClickStats holds aggregated click counts for page-type campaigns,
// keyed by (campaign_id, vid). Flushed from memory to the database
// periodically to avoid per-request writes on a single SQLite connection.
type PageClickStats struct {
	Id          int64     `json:"id" gorm:"primaryKey"`
	CampaignId  int64     `json:"campaign_id"`
	IP          string    `json:"ip"`
	Vid         string    `json:"vid"`
	ClickCount  int64     `json:"click_count"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// TableName fixes the table name for GORM.
func (PageClickStats) TableName() string {
	return "page_click_stats"
}

// clickEntry is the in-memory accumulator for a single (campaign_id, vid) pair.
type clickEntry struct {
	count      int64
	ip         string // most recently seen IP
	lastSeenAt time.Time
}

// clickKey is the composite key for the in-memory counter map.
type clickKey struct {
	campaignID int64
	vid        string
}

// ClickCounter is a thread-safe in-memory counter for page open events.
// It accumulates click counts keyed by (campaign_id, vid) and periodically
// flushes them to the page_click_stats table via upsert.
//
// NOTE: If the process is killed non-gracefully (SIGKILL, OOM, power loss),
// the accumulated counts since the last flush are lost. This is an accepted
// trade-off documented in the codebase.
var ClickCounter = &clickCounter{
	entries: make(map[clickKey]*clickEntry),
}

type clickCounter struct {
	mu      sync.Mutex
	entries map[clickKey]*clickEntry
}

// Incr records a single page open event in memory. It is safe for concurrent
// use. This function does NOT touch the database.
func (cc *clickCounter) Incr(campaignID int64, vid, ip string) {
	if vid == "" {
		return
	}
	key := clickKey{campaignID: campaignID, vid: vid}
	now := time.Now().UTC()

	cc.mu.Lock()
	defer cc.mu.Unlock()

	entry, ok := cc.entries[key]
	if !ok {
		cc.entries[key] = &clickEntry{
			count:      1,
			ip:         ip,
			lastSeenAt: now,
		}
		return
	}
	entry.count++
	// Always update IP to the most recently seen one.
	if ip != "" {
		entry.ip = ip
	}
	entry.lastSeenAt = now
}

// snapshot swaps out the current entries map and returns it, resetting the
// counter to an empty state. This minimises the lock hold time.
func (cc *clickCounter) snapshot() map[clickKey]*clickEntry {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	snap := cc.entries
	cc.entries = make(map[clickKey]*clickEntry)
	return snap
}

// FlushToDB writes accumulated click counts to the database using upsert
// semantics (click_count += value). Individual row failures are logged but
// do not abort the entire flush.
func (cc *clickCounter) FlushToDB() error {
	snap := cc.snapshot()
	if len(snap) == 0 {
		return nil
	}

	now := time.Now().UTC()
	sqlDB, err := db.DB()
	if err != nil {
		log.Errorf("click_counter: failed to get sql.DB: %v", err)
		return err
	}

	for key, entry := range snap {
		// Use raw SQL for upsert to support both SQLite and MySQL with
		// database-specific conflict resolution.
		_, err := sqlDB.Exec(
			`INSERT INTO page_click_stats (campaign_id, ip, vid, click_count, first_seen_at, last_seen_at)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(campaign_id, vid) DO UPDATE SET
			   click_count = click_count + excluded.click_count,
			   ip = excluded.ip,
			   last_seen_at = excluded.last_seen_at`,
			key.campaignID, entry.ip, key.vid, entry.count, now, entry.lastSeenAt,
		)
		if err != nil {
			log.Errorf("click_counter: flush failed for campaign=%d vid=%s: %v",
				key.campaignID, key.vid, err)
			// Continue with remaining entries — don't lose the whole batch.
		}
	}
	return nil
}

// FlushToDBMySQL writes accumulated click counts using MySQL-specific upsert
// syntax. This is used when the database backend is MySQL.
func (cc *clickCounter) FlushToDBMySQL() error {
	snap := cc.snapshot()
	if len(snap) == 0 {
		return nil
	}

	now := time.Now().UTC()
	sqlDB, err := db.DB()
	if err != nil {
		log.Errorf("click_counter: failed to get sql.DB: %v", err)
		return err
	}

	for key, entry := range snap {
		_, err := sqlDB.Exec(
			`INSERT INTO page_click_stats (campaign_id, ip, vid, click_count, first_seen_at, last_seen_at)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
			   click_count = click_count + VALUES(click_count),
			   ip = VALUES(ip),
			   last_seen_at = VALUES(last_seen_at)`,
			key.campaignID, entry.ip, key.vid, entry.count, now, entry.lastSeenAt,
		)
		if err != nil {
			log.Errorf("click_counter: flush failed for campaign=%d vid=%s: %v",
				key.campaignID, key.vid, err)
		}
	}
	return nil
}

// StartFlushLoop runs a periodic flush of in-memory click counts to the
// database. It blocks until the stop channel is closed.
func (cc *clickCounter) StartFlushLoop(interval time.Duration, stop <-chan struct{}) {
	log.Infof("click_counter: starting flush loop with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cc.FlushToDB(); err != nil {
				log.Errorf("click_counter: periodic flush error: %v", err)
			}
		case <-stop:
			log.Info("click_counter: flush loop stopped")
			return
		}
	}
}

// StartFlushLoopMySQL is the MySQL variant of StartFlushLoop.
func (cc *clickCounter) StartFlushLoopMySQL(interval time.Duration, stop <-chan struct{}) {
	log.Infof("click_counter: starting MySQL flush loop with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cc.FlushToDBMySQL(); err != nil {
				log.Errorf("click_counter: periodic flush error: %v", err)
			}
		case <-stop:
			log.Info("click_counter: flush loop stopped")
			return
		}
	}
}

// GenerateVisitorID generates a cryptographically random visitor identifier
// suitable for use as a cookie value.
func GenerateVisitorID() (string, error) {
	return GenerateReportSalt() // same crypto/rand + hex pattern
}

// GetPageClickStats returns the click stats record for a specific
// (campaign_id, vid) pair, or nil if not found.
func GetPageClickStats(campaignID int64, vid string) (*PageClickStats, error) {
	var pcs PageClickStats
	err := db.Where("campaign_id = ? AND vid = ?", campaignID, vid).First(&pcs).Error
	if err != nil {
		return nil, err
	}
	return &pcs, nil
}

// GetPageClickStatsByVid returns all click stats for a campaign keyed by vid.
func GetPageClickStatsByVid(campaignID int64) (map[string]*PageClickStats, error) {
	var all []PageClickStats
	if err := db.Where("campaign_id = ?", campaignID).Find(&all).Error; err != nil {
		return nil, err
	}
	m := make(map[string]*PageClickStats, len(all))
	for i := range all {
		m[all[i].Vid] = &all[i]
	}
	return m, nil
}
