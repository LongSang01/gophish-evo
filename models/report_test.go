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

	re, err := SaveReportExt(camp.Id, SourceTypeClient, Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:ff"}, rc, "127.0.0.1", "test-ua")
	c.Assert(err, check.IsNil)
	c.Assert(re, check.NotNil)
	c.Assert(re.IP, check.Equals, "127.0.0.1")
	c.Assert(re.Source, check.Equals, SourceTypeClient)
}

func (s *ModelsSuite) TestSaveReportExtDedup(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	data := Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:ff"}
	re1, err := SaveReportExt(camp.Id, SourceTypeClient, data, rc, "127.0.0.1", "ua")
	c.Assert(err, check.IsNil)
	c.Assert(re1, check.NotNil)

	// Same MAC → should be skipped (dedup).
	re2, err := SaveReportExt(camp.Id, SourceTypeClient, Map{"ip": "5.6.7.8", "mac": "aa:bb:cc:dd:ee:ff"}, rc, "127.0.0.1", "ua")
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

	SaveReportExt(camp.Id, SourceTypeClient, Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:01"}, rc, "127.0.0.1", "ua")
	SaveReportExt(camp.Id, SourceTypeClient, Map{"ip": "1.2.3.5", "mac": "aa:bb:cc:dd:ee:02"}, rc, "127.0.0.1", "ua")

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

	SaveReportExt(camp.Id, SourceTypePage, Map{"username": "a"}, rc, "127.0.0.1", "ua")
	SaveReportExt(camp.Id, SourceTypePage, Map{"username": "b"}, rc, "127.0.0.1", "ua")

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
	inserted, err := SaveReportExtBatch(camp.Id, SourceTypeClient, records, rc, "127.0.0.1", "ua")
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
	inserted, err := SaveReportExtBatch(camp.Id, SourceTypeClient, records, rc, "127.0.0.1", "ua")
	c.Assert(err, check.IsNil)
	c.Assert(inserted, check.Equals, 1)
}

func (s *ModelsSuite) TestSaveReportExtBatchEmptyRecords(c *check.C) {
	camp := s.createClientCampaignForReport(c)
	rc, err := GetCampaignReportConfig(&camp)
	c.Assert(err, check.IsNil)

	inserted, err := SaveReportExtBatch(camp.Id, SourceTypeClient, []Map{}, rc, "127.0.0.1", "ua")
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
		SaveReportExt(camp.Id, SourceTypeClient, Map{
			"ip":  fmt.Sprintf("10.0.0.%d", i),
			"mac": fmt.Sprintf("aa:bb:cc:dd:ee:%02x", i),
		}, rc, "127.0.0.1", "ua")
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
		SaveReportExt(camp.Id, SourceTypeClient, Map{
			"ip":  fmt.Sprintf("10.0.0.%d", i),
			"mac": fmt.Sprintf("aa:bb:cc:dd:ee:%02x", i),
		}, rc, "127.0.0.1", "ua")
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
	SaveReportExt(camp.Id, SourceTypeClient, Map{"ip": "1.2.3.4", "mac": "aa:bb:cc:dd:ee:ff"}, rc, "127.0.0.1", "ua")

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
