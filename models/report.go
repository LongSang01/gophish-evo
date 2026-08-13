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
	Source     string    `json:"source"`
	DataJSON   string    `json:"-"`
	Data       Map       `json:"data" gorm:"-"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	DedupValue string    `json:"-"`
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

// SaveReportExt validates that the campaign supports reporting and inserts a
// single report record, skipping insert when the dedup key already exists.
func SaveReportExt(campaignID int64, source string, data Map, rc *ReportConfig, ip, ua string) (*ReportExt, error) {
	re := &ReportExt{
		CampaignId: campaignID,
		Source:     source,
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
	if rc != nil && rc.DedupKey != "" {
		if v, ok := data[rc.DedupKey]; ok && v != nil {
			re.DedupValue = fmt.Sprintf("%v", v)
		} else {
			// The dedup field is not present in the reported data (e.g. a
			// page campaign whose form has no "mac" field).  Use a
			// cryptographically random value so every submission is stored
			// instead of being rejected by the unique index.
			buf := make([]byte, 8)
			rand.Read(buf)
			re.DedupValue = fmt.Sprintf("auto-%d-%x", campaignID, buf)
		}
	} else {
		// No dedup key configured (e.g. page campaigns).  Generate a unique
		// value so every submission is stored independently.
		buf := make([]byte, 8)
		rand.Read(buf)
		re.DedupValue = fmt.Sprintf("auto-%d-%x", campaignID, buf)
	}
	if err := db.Create(re).Error; err != nil {
		if isDuplicateKeyError(err) {
			// Idempotent retry - the record already exists.
			return nil, nil
		}
		return nil, err
	}
	return re, nil
}

// SaveReportExtBatch inserts a batch of report records. Records whose
// (campaign_id, source, dedup_value) already exist are skipped so concurrent
// client retries do not produce duplicates.
func SaveReportExtBatch(campaignID int64, source string, records []Map, rc *ReportConfig, ip, ua string) (int, error) {
	inserted := 0
	for _, data := range records {
		if len(data) == 0 {
			continue
		}
		re, err := SaveReportExt(campaignID, source, data, rc, ip, ua)
		if err != nil {
			return inserted, err
		}
		if re != nil {
			inserted++
		}
	}
	return inserted, nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}

// GetCampaignReports returns the report records for a campaign, optionally
// paginated, ordered newest first.
func GetCampaignReports(campaignID int64, pp PageParams) ([]ReportExt, int64, error) {
	reports := []ReportExt{}
	var total int64
	query := db.Table("reports_ext").Where("campaign_id=?", campaignID)
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
	err := db.Table("reports_ext").Where("campaign_id=?", campaignID).Count(&total).Error
	return total, err
}

// GetPageCampaignByPath returns the fixed-page type campaign whose configured
// URL path matches the given request path. Only page type activities are
// served at a fixed URL without a rid parameter. The query filters by
// source_type in the database so we only load the small set of page campaigns
// rather than scanning all campaigns.
func GetPageCampaignByPath(path string) (*Campaign, error) {
	cs := []Campaign{}
	if err := db.Where("source_type=?", SourceTypePage).Find(&cs).Error; err != nil {
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
	err := db.Table("campaigns").
		Select("id, user_id, page_id, status, url, source_type, report_config_json, report_salt").
		Where("id=? AND status=?", id, CampaignInProgress).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}
