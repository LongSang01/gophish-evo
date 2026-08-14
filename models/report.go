package models

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"gorm.io/gorm"
)

// Activity source types
const (
	SourceTypeEmail  string = "email"
	SourceTypeClient string = "client"
	SourceTypePage   string = "page"
)

// Report field types. These are hints consumed by the frontend (input type,
// placeholder, keyboard); the backend performs no format validation and
// stores whatever the client reports.
const (
	FieldTypeIP       string = "ip"
	FieldTypeMAC      string = "mac"
	FieldTypeUsername string = "username"
	FieldTypeHostname string = "hostname"
	FieldTypeCustom   string = "custom"
)

// DefaultClientFields are the default collectable fields offered when a
// client-type activity is created.
var DefaultClientFields = []ReportField{
	{Key: "ip", Label: "IP地址", Type: FieldTypeIP},
	{Key: "mac", Label: "MAC地址", Type: FieldTypeMAC},
	{Key: "username", Label: "用户名", Type: FieldTypeUsername},
	{Key: "hostname", Label: "主机名", Type: FieldTypeHostname},
}

// ErrReportKeyInvalid indicates the provided report authentication key is
// invalid.
var ErrReportKeyInvalid = errors.New("Invalid report key")

// ErrReportCampaignNotFound indicates the reported campaign was not found or
// is not a reporting-capable activity.
var ErrReportCampaignNotFound = errors.New("Campaign not found or not a report activity")

// ErrReportConfigInvalid indicates the stored report config is malformed.
var ErrReportConfigInvalid = errors.New("Invalid report config")

// ReportField describes a single collectable field for client/page type
// activities.
type ReportField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// ReportConfig is the per-campaign configuration used to render the client /
// fixed-page collection form.
type ReportConfig struct {
	Fields   []ReportField `json:"fields"`
	DedupKey string        `json:"dedup_key"`
}

// NewReportConfig returns a default report configuration for a client-type
// activity, seeded with the default client fields and MAC dedup key.
func NewReportConfig() *ReportConfig {
	return &ReportConfig{
		Fields:   DefaultClientFields,
		DedupKey: "mac",
	}
}

// NewPageReportConfig returns a default report configuration for a page-type
// activity.  Page forms are user-authored so we cannot assume any specific
// field names; dedup is disabled by default (every submission is stored).
func NewPageReportConfig() *ReportConfig {
	return &ReportConfig{
		Fields:   []ReportField{},
		DedupKey: "",
	}
}

// UnmarshalReportConfig parses a report_config_json string into a
// ReportConfig, falling back to the default configuration when empty.
func UnmarshalReportConfig(raw string) (*ReportConfig, error) {
	rc := NewReportConfig()
	if strings.TrimSpace(raw) == "" {
		return rc, nil
	}
	if err := json.Unmarshal([]byte(raw), rc); err != nil {
		return nil, err
	}
	return rc, nil
}

// Marshal returns the JSON serialization of the report configuration.
func (rc *ReportConfig) Marshal() string {
	b, err := json.Marshal(rc)
	if err != nil {
		log.Error(err)
		return "{}"
	}
	return string(b)
}

// Lookup returns the field definition for the given key, or nil.
func (rc *ReportConfig) Lookup(key string) *ReportField {
	for i := range rc.Fields {
		if rc.Fields[i].Key == key {
			return &rc.Fields[i]
		}
	}
	return nil
}

// ReportExt is a record of reported data from the client or fixed-page
// modules, stored in the reports_ext table.
type ReportExt struct {
	Id         int64     `json:"id"`
	CampaignId int64     `json:"campaign_id"`
	Vid        string    `json:"vid"` // Visitor ID for page-type campaigns
	DataJSON   string    `json:"-"`
	Data       Map       `json:"data" gorm:"-"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

// Map is a generic JSON object used for the dynamic report data.
type Map map[string]interface{}

// TableName fixes the table name to the reports_ext table created by the
// migration, rather than GORM's pluralization of the struct name.
func (ReportExt) TableName() string {
	return "reports_ext"
}

// GenerateReportSalt returns a cryptographically random hex salt used to
// derive the per-campaign report authentication key.
func GenerateReportSalt() (string, error) {
	k := make([]byte, 16)
	if _, err := rand.Read(k); err != nil {
		return "", err
	}
	return hex.EncodeToString(k), nil
}

// ReportKey returns the authentication key for the given campaign id and
// salt: hex(md5("<campaign_id>-<salt>")).
func ReportKey(campaignID int64, salt string) string {
	sum := md5.Sum(fmt.Appendf(nil, "%d-%s", campaignID, salt))
	return hex.EncodeToString(sum[:])
}

// ValidateReportKey compares the supplied key against the derived key for the
// campaign. A non-empty salt is required; empty salt (legacy activities) is
// rejected.
func ValidateReportKey(campaignID int64, salt, key string) bool {
	if salt == "" || key == "" {
		return false
	}
	return ReportKey(campaignID, salt) == key
}

// GetCampaignReportConfig loads the report configuration for a campaign.
func GetCampaignReportConfig(c *Campaign) (*ReportConfig, error) {
	rc, err := UnmarshalReportConfig(c.ReportConfigJSON)
	if err != nil {
		log.Error(err)
		return nil, ErrReportConfigInvalid
	}
	return rc, nil
}

// SaveReportExt inserts a single report record. For client-type campaigns,
// duplicate detection is performed at the application level by checking
// whether a record with the same dedup key value already exists.
// vid is the visitor identifier for page-type campaigns (may be empty for
// non-page sources).
func SaveReportExt(campaignID int64, data Map, rc *ReportConfig, ip, ua, vid string) (*ReportExt, error) {
	re := &ReportExt{
		CampaignId: campaignID,
		Vid:        vid,
		Data:       data,
		IP:         ip,
		UserAgent:  ua,
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	re.DataJSON = string(dataJSON)
	re.CreatedAt = time.Now().UTC()

	// For client-type campaigns with a dedup key configured, check for
	// existing records with the same dedup value before inserting.
	if rc != nil && rc.DedupKey != "" {
		if v, ok := data[rc.DedupKey]; ok && v != nil {
			dedupVal := fmt.Sprintf("%v", v)
			var count int64
			if err := db.Table("reports_ext").Where(
				"campaign_id=? AND data_json LIKE ?",
				campaignID, "%\""+rc.DedupKey+"\":"+fmt.Sprintf("%q", dedupVal)+"%").
				Count(&count).Error; err != nil {
				return nil, err
			}
			if count > 0 {
				return nil, nil // duplicate, skip
			}
		}
	}

	if err := db.Create(re).Error; err != nil {
		return nil, err
	}
	return re, nil
}

// SaveReportExtBatch inserts a batch of report records. For client-type
// campaigns with a dedup key, duplicates are detected at the application
// level. vid is the visitor identifier for page-type campaigns.
func SaveReportExtBatch(campaignID int64, records []Map, rc *ReportConfig, ip, ua, vid string) (int, error) {
	inserted := 0
	for _, data := range records {
		if len(data) == 0 {
			continue
		}
		re, err := SaveReportExt(campaignID, data, rc, ip, ua, vid)
		if err != nil {
			return inserted, err
		}
		if re != nil {
			inserted++
		}
	}
	return inserted, nil
}

// GetCampaignReports returns the report records for a campaign, optionally
// paginated, ordered newest first.
func GetCampaignReports(campaignID int64, pp PageParams) ([]ReportExt, int64, error) {
	reports := []ReportExt{}
	var total int64
	query := readDB().Table("reports_ext").Where("campaign_id=?", campaignID)
	if pp.Valid() {
		if err := query.Count(&total).Error; err != nil {
			log.Error(err)
			return reports, 0, err
		}
	}
	query = query.Order("id DESC")
	if pp.Valid() {
		query = query.Limit(pp.PageSize).Offset(pp.Offset())
	}
	if err := query.Find(&reports).Error; err != nil {
		log.Error(err)
		return reports, 0, err
	}
	for i := range reports {
		var m Map
		if err := json.Unmarshal([]byte(reports[i].DataJSON), &m); err != nil {
			m = Map{}
		}
		reports[i].Data = m
	}
	if !pp.Valid() {
		total = int64(len(reports))
	}
	return reports, total, nil
}

// DeleteCampaignReports removes all report records belonging to a campaign.
func DeleteCampaignReports(campaignID int64) error {
	err := db.Where("campaign_id=?", campaignID).Delete(&ReportExt{}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetCampaignReportCount returns the number of report records for a campaign.
func GetCampaignReportCount(campaignID int64) (int64, error) {
	var total int64
	err := readDB().Table("reports_ext").Where("campaign_id=?", campaignID).Count(&total).Error
	return total, err
}

// ReportSummaryRow represents a single row in the aggregated report view.
type ReportSummaryRow struct {
	Source          string    `json:"source"`
	Vid             string    `json:"vid"`
	IP              string    `json:"ip"`
	UserAgent       string    `json:"user_agent"`
	Submitted       bool      `json:"submitted"`
	SubmissionCount int64     `json:"submission_count"`
	ClickCount      int64     `json:"click_count"`
	LastClickAt     time.Time `json:"last_click_at,omitempty"`
	ReportExtId     int64     `json:"report_ext_id,omitempty"` // non-zero for submitted visitors
	DataJSON        string    `json:"-"`
	Data            Map       `json:"data" gorm:"-"`
	CreatedAt       time.Time `json:"created_at"` // for submitted: last submission time; for click-only: last click time
	FirstSeenAt     time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt      time.Time `json:"last_seen_at,omitempty"`
}

// GetCampaignReportSummary returns the aggregated report view for a campaign.
// It merges submitted reports (reports_ext) with click statistics
// (page_click_stats) to produce a unified view:
//
//   - Submitted visitors (have reports_ext records): one row per vid with
//     ClickCount = total page opens (from page_click_stats).
//   - Click-only visitors (have page_click_stats but no reports_ext):
//     one row per vid with SubmissionCount = 0.
//   - Legacy records (no vid, vid == ”): each record is its own row.
//
// Results are ordered: submitted visitors first (sorted by submission time),
// then click-only visitors (sorted by last click time).
func GetCampaignReportSummary(campaignID int64, pp PageParams) ([]ReportSummaryRow, int64, error) {
	rows := []ReportSummaryRow{}
	var total int64

	// 1. Load all reports_ext for this campaign.
	var reports []ReportExt
	if err := readDB().Where("campaign_id=?", campaignID).
		Order("id ASC").Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	// 2. Load all page_click_stats for this campaign.
	var clickStats []PageClickStats
	if err := readDB().Where("campaign_id=?", campaignID).Find(&clickStats).Error; err != nil {
		return nil, 0, err
	}

	// Build a map from vid → click stats.
	clickMap := make(map[string]*PageClickStats, len(clickStats))
	for i := range clickStats {
		clickMap[clickStats[i].Vid] = &clickStats[i]
	}

	// 3. Group reports by vid (non-empty vid). Reports with empty vid
	//    (legacy data) are each their own group.
	type reportGroup struct {
		reports []ReportExt
		vid     string
	}
	groupByVid := map[string]*reportGroup{}
	var legacyGroups []*reportGroup

	for _, r := range reports {
		if r.Vid == "" {
			// Legacy: each record is its own row.
			legacyGroups = append(legacyGroups, &reportGroup{reports: []ReportExt{r}, vid: ""})
			continue
		}
		g, ok := groupByVid[r.Vid]
		if !ok {
			g = &reportGroup{vid: r.Vid}
			groupByVid[r.Vid] = g
		}
		g.reports = append(g.reports, r)
	}

	// 4. Build summary rows for submitted visitors (non-empty vid).
	//    Use the LAST report as the representative row (most recent data).
	submittedVids := make(map[string]bool)
	for vid, g := range groupByVid {
		last := g.reports[len(g.reports)-1]
		var m Map
		if err := json.Unmarshal([]byte(last.DataJSON), &m); err != nil {
			m = Map{}
		}
		totalClicks := int64(len(g.reports)) // at least as many clicks as submissions
		if cs, ok := clickMap[vid]; ok {
			totalClicks = cs.ClickCount
		}

		var lastClickAt time.Time
		if cs, ok := clickMap[vid]; ok {
			lastClickAt = cs.LastSeenAt
		}

		rows = append(rows, ReportSummaryRow{
			Source:          SourceTypePage,
			Vid:             vid,
			IP:              last.IP,
			UserAgent:       last.UserAgent,
			Submitted:       true,
			SubmissionCount: int64(len(g.reports)),
			ClickCount:      totalClicks,
			LastClickAt:     lastClickAt,
			ReportExtId:     last.Id,
			DataJSON:        last.DataJSON,
			Data:            m,
			CreatedAt:       last.CreatedAt,
			FirstSeenAt:     last.CreatedAt,
			LastSeenAt:      last.CreatedAt,
		})
		submittedVids[vid] = true
	}

	// 5. Build summary rows for legacy submissions (empty vid).
	for _, g := range legacyGroups {
		r := g.reports[0]
		var m Map
		if err := json.Unmarshal([]byte(r.DataJSON), &m); err != nil {
			m = Map{}
		}
		rows = append(rows, ReportSummaryRow{
			Source:          SourceTypePage,
			Vid:             "",
			IP:              r.IP,
			UserAgent:       r.UserAgent,
			Submitted:       true,
			SubmissionCount: 1,
			ClickCount:      0, // no click stats for legacy records
			ReportExtId:     r.Id,
			DataJSON:        r.DataJSON,
			Data:            m,
			CreatedAt:       r.CreatedAt,
			FirstSeenAt:     r.CreatedAt,
			LastSeenAt:      r.CreatedAt,
		})
	}

	// 6. Build summary rows for click-only visitors (in clickMap but no reports).
	for vid, cs := range clickMap {
		if submittedVids[vid] {
			continue
		}
		rows = append(rows, ReportSummaryRow{
			Source:          SourceTypePage,
			Vid:             vid,
			Submitted:       false,
			SubmissionCount: 0,
			ClickCount:      cs.ClickCount,
			LastClickAt:     cs.LastSeenAt,
			Data:            Map{},
			CreatedAt:       cs.LastSeenAt,
			FirstSeenAt:     cs.FirstSeenAt,
			LastSeenAt:      cs.LastSeenAt,
		})
	}

	// 7. Sort: submitted first (by last submission time DESC), then
	//    click-only (by last click time DESC).
	sortSummaryRows(rows)

	total = int64(len(rows))

	// 8. Apply pagination manually.
	if pp.Valid() {
		start := pp.Offset()
		end := start + pp.PageSize
		if start > len(rows) {
			start = len(rows)
		}
		if end > len(rows) {
			end = len(rows)
		}
		rows = rows[start:end]
	}

	return rows, total, nil
}

// sortSummaryRows sorts the summary rows: submitted first (newest first),
// then click-only (newest first).
func sortSummaryRows(rows []ReportSummaryRow) {
	// Simple insertion sort — the dataset is typically small.
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 {
			swap := false
			a, b := rows[j-1], rows[j]
			// Submitted rows come before non-submitted.
			if a.Submitted != b.Submitted {
				if !a.Submitted && b.Submitted {
					swap = true
				}
			} else {
				// Within the same group, sort by most recent time.
				timeA := a.LastSeenAt
				timeB := b.LastSeenAt
				if a.Submitted {
					timeA = a.FirstSeenAt
					timeB = b.FirstSeenAt
				}
				if timeA.Before(timeB) {
					swap = true
				}
			}
			if !swap {
				break
			}
			rows[j-1], rows[j] = rows[j], rows[j-1]
			j--
		}
	}
}

// GetPageCampaignByPath returns the fixed-page type campaign whose configured
// URL path matches the given request path. Only page type activities are
// served at a fixed URL without a rid parameter. The query filters by
// source_type in the database so we only load the small set of page campaigns
// rather than scanning all campaigns.
func GetPageCampaignByPath(path string) (*Campaign, error) {
	cs := []Campaign{}
	if err := readDB().Where("source_type=?", SourceTypePage).Find(&cs).Error; err != nil {
		return nil, err
	}
	for i := range cs {
		u, err := url.Parse(cs[i].URL)
		if err != nil {
			continue
		}
		if u.Path == path || (u.Path == "" && path == "/") {
			return &cs[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// GetCampaignForReport loads the minimal fields needed to authenticate and
// store report data for a campaign, without loading results/groups/stats.
// Only campaigns that are InProgress are returned; completed campaigns are
// treated as not found so no new data can be submitted.
func GetCampaignForReport(id int64) (*Campaign, error) {
	c := Campaign{}
	err := readDB().Table("campaigns").
		Select("id, user_id, page_id, status, url, source_type, report_config_json, report_salt").
		Where("id=? AND status=?", id, CampaignInProgress).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}
