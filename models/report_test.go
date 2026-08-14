package models

import (
	"encoding/json"
	"fmt"

	check "gopkg.in/check.v1"
)

// ---------------------------------------------------------------------------
// ReportKey / ValidateReportKey
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestReportKeyDeterministic(c *check.C) {
	k1 := ReportKey(42, "salt-value")
	k2 := ReportKey(42, "salt-value")
	c.Assert(k1, check.Equals, k2)
	c.Assert(len(k1) > 0, check.Equals, true)
}

func (s *ModelsSuite) TestReportKeyDifferentInputs(c *check.C) {
	k1 := ReportKey(1, "salt")
	k2 := ReportKey(2, "salt")
	k3 := ReportKey(1, "other")
	c.Assert(k1 != k2, check.Equals, true)
	c.Assert(k1 != k3, check.Equals, true)
}

func (s *ModelsSuite) TestValidateReportKeyValid(c *check.C) {
	c.Assert(ValidateReportKey(10, "my-salt", ReportKey(10, "my-salt")), check.Equals, true)
}

func (s *ModelsSuite) TestValidateReportKeyInvalid(c *check.C) {
	c.Assert(ValidateReportKey(10, "my-salt", "wrong"), check.Equals, false)
}

func (s *ModelsSuite) TestValidateReportKeyEmptySalt(c *check.C) {
	c.Assert(ValidateReportKey(10, "", ReportKey(10, "")), check.Equals, false)
}

func (s *ModelsSuite) TestValidateReportKeyEmptyKey(c *check.C) {
	c.Assert(ValidateReportKey(10, "salt", ""), check.Equals, false)
}

// ---------------------------------------------------------------------------
// GenerateReportSalt
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestGenerateReportSalt(c *check.C) {
	salt, err := GenerateReportSalt()
	c.Assert(err, check.IsNil)
	c.Assert(len(salt) > 0, check.Equals, true)
	// Two calls should produce different salts.
	salt2, _ := GenerateReportSalt()
	c.Assert(salt != salt2, check.Equals, true)
}

// ---------------------------------------------------------------------------
// ReportConfig marshal / unmarshal
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestReportConfigMarshalRoundTrip(c *check.C) {
	rc := &ReportConfig{
		Fields: []ReportField{
			{Key: "ip", Label: "IP", Type: FieldTypeIP, Required: true},
			{Key: "mac", Label: "MAC", Type: FieldTypeMAC},
		},
		DedupKey: "mac",
	}
	raw := rc.Marshal()
	c.Assert(len(raw) > 0, check.Equals, true)

	rc2, err := UnmarshalReportConfig(raw)
	c.Assert(err, check.IsNil)
	c.Assert(rc2.DedupKey, check.Equals, "mac")
	c.Assert(len(rc2.Fields), check.Equals, 2)
	c.Assert(rc2.Fields[0].Key, check.Equals, "ip")
	c.Assert(rc2.Fields[0].Required, check.Equals, true)
}

func (s *ModelsSuite) TestReportConfigUnmarshalEmpty(c *check.C) {
	rc, err := UnmarshalReportConfig("")
	c.Assert(err, check.IsNil)
	// Should fall back to default client fields.
	c.Assert(len(rc.Fields) > 0, check.Equals, true)
	c.Assert(rc.DedupKey, check.Equals, "mac")
}

func (s *ModelsSuite) TestReportConfigLookup(c *check.C) {
	rc := NewReportConfig()
	f := rc.Lookup("mac")
	c.Assert(f, check.NotNil)
	c.Assert(f.Type, check.Equals, FieldTypeMAC)
	c.Assert(rc.Lookup("nonexistent"), check.IsNil)
}

func (s *ModelsSuite) TestReportConfigUnmarshalInvalidJSON(c *check.C) {
	_, err := UnmarshalReportConfig("{invalid json")
	c.Assert(err, check.NotNil)
}

// ---------------------------------------------------------------------------
// SaveReportExt + dedup
// ---------------------------------------------------------------------------

func (s *ModelsSuite) createClientCampaignForReport(c *check.C) Campaign {
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypeClient
	camp.ReportConfig = &ReportConfig{
		Fields:   DefaultClientFields,
		DedupKey: "mac",
	}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)
	return camp
}

func (s *ModelsSuite) TestSaveReportExtBasic(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	re, err := SaveReportExt(camp.Id, Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:ff"}, rc, "127.0.0.1", "test-ua", "")
	c.Assert(err, check.IsNil)
	c.Assert(re, check.NotNil)
	c.Assert(re.IP, check.Equals, "127.0.0.1")
}

func (s *ModelsSuite) TestSaveReportExtDedup(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	data := Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:ff"}
	re1, err := SaveReportExt(camp.Id, data, rc, "127.0.0.1", "ua", "")
	c.Assert(err, check.IsNil)
	c.Assert(re1, check.NotNil)

	// Same MAC → should be skipped (dedup).
	re2, err := SaveReportExt(camp.Id, Map{"ip": "5.6.7.8", "mac": "aa:bb:cc:dd:ee:ff"}, rc, "127.0.0.1", "ua", "")
	c.Assert(err, check.IsNil)
	c.Assert(re2, check.IsNil) // nil means skipped

	total, err := GetCampaignReportCount(camp.Id)
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(1))
}

func (s *ModelsSuite) TestSaveReportExtDifferentDedup(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	SaveReportExt(camp.Id, Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:01"}, rc, "127.0.0.1", "ua", "")
	SaveReportExt(camp.Id, Map{"ip": "1.2.3.5", "mac": "aa:bb:cc:dd:ee:02"}, rc, "127.0.0.1", "ua", "")

	total, err := GetCampaignReportCount(camp.Id)
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(2))
}

func (s *ModelsSuite) TestSaveReportExtNoDedupKey(c *check.C) {
	// Page campaigns with no dedup key → every submission is stored.
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypePage
	camp.ReportConfig = &ReportConfig{
		Fields:   []ReportField{},
		DedupKey: "",
	}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	SaveReportExt(camp.Id, Map{"username": "a"}, rc, "127.0.0.1", "ua", "vid-test-1")
	SaveReportExt(camp.Id, Map{"username": "b"}, rc, "127.0.0.1", "ua", "vid-test-2")

	total, err := GetCampaignReportCount(camp.Id)
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(2))
}

// ---------------------------------------------------------------------------
// SaveReportExtBatch
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestSaveReportExtBatch(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	records := []Map{
		{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:01"},
		{"ip": "1.2.3.5", "mac": "aa:bb:cc:dd:ee:02"},
		{"ip": "1.2.3.6", "mac": "aa:bb:cc:dd:ee:03"},
	}
	inserted, err := SaveReportExtBatch(camp.Id, records, rc, "127.0.0.1", "ua", "")
	c.Assert(err, check.IsNil)
	c.Assert(inserted, check.Equals, 3)

	total, err := GetCampaignReportCount(camp.Id)
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(3))
}

func (s *ModelsSuite) TestSaveReportExtBatchWithDuplicates(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	records := []Map{
		{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:01"},
		{"ip": "1.2.3.5", "mac": "aa:bb:cc:dd:ee:01"}, // dup MAC
	}
	inserted, err := SaveReportExtBatch(camp.Id, records, rc, "127.0.0.1", "ua", "")
	c.Assert(err, check.IsNil)
	c.Assert(inserted, check.Equals, 1)
}

func (s *ModelsSuite) TestSaveReportExtBatchEmptyRecords(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	inserted, err := SaveReportExtBatch(camp.Id, []Map{}, rc, "127.0.0.1", "ua", "")
	c.Assert(err, check.IsNil)
	c.Assert(inserted, check.Equals, 0)
}

// ---------------------------------------------------------------------------
// GetCampaignReports + pagination
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestGetCampaignReportsEmpty(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	reports, total, err := GetCampaignReports(camp.Id, PageParams{})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(0))
	c.Assert(len(reports), check.Equals, 0)
}

func (s *ModelsSuite) TestGetCampaignReportsWithData(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, _ := GetCampaignReportConfig(&camp)

	for i := 0; i < 5; i++ {
		SaveReportExt(camp.Id, Map{
			"ip":  fmt.Sprintf("10.0.0.%d", i),
			"mac": fmt.Sprintf("aa:bb:cc:dd:ee:%02x", i),
		}, rc, "127.0.0.1", "ua", "")
	}

	reports, total, err := GetCampaignReports(camp.Id, PageParams{})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(5))
	c.Assert(len(reports), check.Equals, 5)
	// Verify data was deserialized.
	c.Assert(reports[0].Data, check.NotNil)
}

func (s *ModelsSuite) TestGetCampaignReportsPagination(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, _ := GetCampaignReportConfig(&camp)

	for i := 0; i < 10; i++ {
		SaveReportExt(camp.Id, Map{
			"ip":  fmt.Sprintf("10.0.0.%d", i),
			"mac": fmt.Sprintf("aa:bb:cc:dd:ee:%02x", i),
		}, rc, "127.0.0.1", "ua", "")
	}

	// Page 1, size 3.
	reports, total, err := GetCampaignReports(camp.Id, PageParams{Page: 1, PageSize: 3})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(10))
	c.Assert(len(reports), check.Equals, 3)

	// Page 4, size 3 → should get 1 remaining.
	reports, total, err = GetCampaignReports(camp.Id, PageParams{Page: 4, PageSize: 3})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(10))
	c.Assert(len(reports), check.Equals, 1)
}

// ---------------------------------------------------------------------------
// DeleteCampaignReports
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestDeleteCampaignReports(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, _ := GetCampaignReportConfig(&camp)
	SaveReportExt(camp.Id, Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:ff"}, rc, "127.0.0.1", "ua", "")

	err := DeleteCampaignReports(camp.Id)
	c.Assert(err, check.IsNil)

	total, err := GetCampaignReportCount(camp.Id)
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(0))
}

// ---------------------------------------------------------------------------
// UnmarshalReportConfig
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestReportConfigJSONSerialization(c *check.C) {
	raw := `{"fields":[{"key":"hostname","label":"Host","type":"hostname","required":true}],"dedup_key":"hostname"}`
	rc, err := UnmarshalReportConfig(raw)
	c.Assert(err, check.IsNil)
	c.Assert(rc.DedupKey, check.Equals, "hostname")
	c.Assert(len(rc.Fields), check.Equals, 1)
	c.Assert(rc.Fields[0].Key, check.Equals, "hostname")

	// Marshal back and verify structure.
	b, err := json.Marshal(rc)
	c.Assert(err, check.IsNil)
	c.Assert(len(b) > 0, check.Equals, true)
}

// ---------------------------------------------------------------------------
// ClickCounter
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestClickCounterIncr(c *check.C) {
	// Reset counter state.
	counter := &clickCounter{entries: make(map[clickKey]*clickEntry)}
	counter.Incr(1, "vid-1")
	counter.Incr(1, "vid-1")
	counter.Incr(1, "vid-1") // same vid, different IP

	counter.mu.Lock()
	entry := counter.entries[clickKey{campaignID: 1, vid: "vid-1"}]
	c.Assert(entry, check.NotNil)
	c.Assert(entry.count, check.Equals, int64(3))
	counter.mu.Unlock()
}

func (s *ModelsSuite) TestClickCounterIncrEmptyVid(c *check.C) {
	counter := &clickCounter{entries: make(map[clickKey]*clickEntry)}
	counter.Incr(1, "") // empty vid should be ignored

	counter.mu.Lock()
	c.Assert(len(counter.entries), check.Equals, 0)
	counter.mu.Unlock()
}

func (s *ModelsSuite) TestClickCounterSnapshot(c *check.C) {
	counter := &clickCounter{entries: make(map[clickKey]*clickEntry)}
	counter.Incr(1, "vid-1")
	counter.Incr(1, "vid-2")

	snap := counter.snapshot()
	c.Assert(len(snap), check.Equals, 2)

	// After snapshot, entries should be empty.
	counter.mu.Lock()
	c.Assert(len(counter.entries), check.Equals, 0)
	counter.mu.Unlock()
}

// ---------------------------------------------------------------------------
// FlushToDB
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestFlushToDB(c *check.C) {
	camp := s.createClientCampaignForReport(c)

	// Use a fresh counter to avoid interference from the global singleton.
	counter := &clickCounter{entries: make(map[clickKey]*clickEntry)}
	counter.Incr(camp.Id, "vid-flush-1")
	counter.Incr(camp.Id, "vid-flush-1")
	counter.Incr(camp.Id, "vid-flush-2")

	err := counter.FlushToDB()
	c.Assert(err, check.IsNil)

	// Verify stats were written.
	stats1, err := GetPageClickStats(camp.Id, "vid-flush-1")
	c.Assert(err, check.IsNil)
	c.Assert(stats1.ClickCount, check.Equals, int64(2))

	stats2, err := GetPageClickStats(camp.Id, "vid-flush-2")
	c.Assert(err, check.IsNil)
	c.Assert(stats2.ClickCount, check.Equals, int64(1))
}

func (s *ModelsSuite) TestFlushToDBAccumulate(c *check.C) {
	camp := s.createClientCampaignForReport(c)

	// First flush.
	counter := &clickCounter{entries: make(map[clickKey]*clickEntry)}
	counter.Incr(camp.Id, "vid-accum")
	counter.Incr(camp.Id, "vid-accum")
	err := counter.FlushToDB()
	c.Assert(err, check.IsNil)

	// Second flush with same vid.
	counter.Incr(camp.Id, "vid-accum")
	counter.Incr(camp.Id, "vid-accum")
	counter.Incr(camp.Id, "vid-accum")
	err = counter.FlushToDB()
	c.Assert(err, check.IsNil)

	// Total should be 2 + 3 = 5.
	stats, err := GetPageClickStats(camp.Id, "vid-accum")
	c.Assert(err, check.IsNil)
	c.Assert(stats.ClickCount, check.Equals, int64(5))
}

// ---------------------------------------------------------------------------
// GetCampaignReportSummary
// ---------------------------------------------------------------------------

func (s *ModelsSuite) TestReportSummarySubmittedWithClicks(c *check.C) {
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypePage
	camp.ReportConfig = &ReportConfig{Fields: []ReportField{}, DedupKey: ""}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)
	rc, _ := GetCampaignReportConfig(&camp)

	// Simulate: visitor submitted 1 form, clicked 5 times total.
	SaveReportExt(camp.Id, Map{"name": "Alice"}, rc, "10.0.0.1", "ua", "vid-alice")

	// Write click stats directly (simulating a flush).
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO page_click_stats (campaign_id, vid, click_count, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, camp.Id, "vid-alice", 5)

	rows, total, err := GetCampaignReportSummary(camp.Id, PageParams{})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(1))
	c.Assert(len(rows), check.Equals, 1)
	c.Assert(rows[0].Submitted, check.Equals, true)
	c.Assert(rows[0].SubmissionCount, check.Equals, int64(1))
	c.Assert(rows[0].ClickCount, check.Equals, int64(5))
	c.Assert(rows[0].Vid, check.Equals, "vid-alice")
}

func (s *ModelsSuite) TestReportSummaryClickOnly(c *check.C) {
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypePage
	camp.ReportConfig = &ReportConfig{Fields: []ReportField{}, DedupKey: ""}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)

	// Only click stats, no submission.
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO page_click_stats (campaign_id, vid, click_count, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, camp.Id, "vid-bob", 3)

	rows, total, err := GetCampaignReportSummary(camp.Id, PageParams{})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(1))
	c.Assert(rows[0].Submitted, check.Equals, false)
	c.Assert(rows[0].SubmissionCount, check.Equals, int64(0))
	c.Assert(rows[0].ClickCount, check.Equals, int64(3))
}

func (s *ModelsSuite) TestReportSummaryMixed(c *check.C) {
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypePage
	camp.ReportConfig = &ReportConfig{Fields: []ReportField{}, DedupKey: ""}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)
	rc, _ := GetCampaignReportConfig(&camp)

	// Visitor A: submitted + clicks.
	SaveReportExt(camp.Id, Map{"name": "Alice"}, rc, "10.0.0.1", "ua", "vid-a")
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO page_click_stats (campaign_id, vid, click_count, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, camp.Id, "vid-a", 3)

	// Visitor B: click-only.
	sqlDB.Exec(`INSERT INTO page_click_stats (campaign_id, vid, click_count, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, camp.Id, "vid-b", 7)

	rows, total, err := GetCampaignReportSummary(camp.Id, PageParams{})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(2))

	// Submitted visitors should come first.
	c.Assert(rows[0].Submitted, check.Equals, true)
	c.Assert(rows[0].Vid, check.Equals, "vid-a")
	c.Assert(rows[0].ClickCount, check.Equals, int64(3))

	c.Assert(rows[1].Submitted, check.Equals, false)
	c.Assert(rows[1].Vid, check.Equals, "vid-b")
	c.Assert(rows[1].ClickCount, check.Equals, int64(7))
}

func (s *ModelsSuite) TestReportSummaryPagination(c *check.C) {
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypePage
	camp.ReportConfig = &ReportConfig{Fields: []ReportField{}, DedupKey: ""}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)
	rc, _ := GetCampaignReportConfig(&camp)

	// Create 5 submitted visitors.
	for i := 0; i < 5; i++ {
		SaveReportExt(camp.Id, Map{"i": i}, rc, "10.0.0.1", "ua", fmt.Sprintf("vid-%d", i))
	}

	// Page 1, size 2.
	rows, total, err := GetCampaignReportSummary(camp.Id, PageParams{Page: 1, PageSize: 2})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(5))
	c.Assert(len(rows), check.Equals, 2)

	// Page 3, size 2 → should get 1 remaining.
	rows, total, err = GetCampaignReportSummary(camp.Id, PageParams{Page: 3, PageSize: 2})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(5))
	c.Assert(len(rows), check.Equals, 1)
}

func (s *ModelsSuite) TestReportSummaryLegacyRecords(c *check.C) {
	camp := s.createCampaignDependencies(c)
	camp.SourceType = SourceTypePage
	camp.ReportConfig = &ReportConfig{Fields: []ReportField{}, DedupKey: ""}
	err := PostCampaign(&camp, camp.UserId)
	c.Assert(err, check.IsNil)
	rc, _ := GetCampaignReportConfig(&camp)

	// Legacy records with empty vid (each is its own row).
	SaveReportExt(camp.Id, Map{"name": "legacy1"}, rc, "10.0.0.1", "ua", "")
	SaveReportExt(camp.Id, Map{"name": "legacy2"}, rc, "10.0.0.2", "ua", "")

	rows, total, err := GetCampaignReportSummary(camp.Id, PageParams{})
	c.Assert(err, check.IsNil)
	c.Assert(total, check.Equals, int64(2))
	c.Assert(rows[0].Submitted, check.Equals, true)
	c.Assert(rows[0].SubmissionCount, check.Equals, int64(1))
	c.Assert(rows[0].ClickCount, check.Equals, int64(0)) // no click stats for legacy
}
