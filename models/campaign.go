package models

import (
	"errors"
	"net/url"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/webhook"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Campaign is a struct representing a created campaign
type Campaign struct {
	Id               int64         `json:"id"`
	UserId           int64         `json:"-"`
	Name             string        `json:"name" sql:"not null"`
	CreatedDate      time.Time     `json:"created_date"`
	LaunchDate       time.Time     `json:"launch_date"`
	SendByDate       time.Time     `json:"send_by_date"`
	CompletedDate    time.Time     `json:"completed_date"`
	TemplateId       int64         `json:"-"`
	Template         Template      `json:"template"`
	PageId           int64         `json:"-"`
	Page             Page          `json:"page"`
	Status           string        `json:"status"`
	Results          []Result      `json:"results,omitempty"`
	Groups           []Group       `json:"groups,omitempty" gorm:"-"`
	Events           []Event       `json:"timeline,omitempty"`
	SMTPId           int64         `json:"-"`
	SMTP             SMTP          `json:"smtp"`
	SMTPs            []SMTP        `json:"smtps,omitempty" gorm:"-"`
	URL              string        `json:"url"`
	Stats            CampaignStats `json:"stats" gorm:"-"`
	SourceType       string        `json:"source_type" gorm:"column:source_type"`
	ReportConfig     *ReportConfig `json:"report_config,omitempty" gorm:"-"`
	ReportSalt       string        `json:"-" gorm:"column:report_salt"`
	ReportConfigJSON string        `json:"-" gorm:"column:report_config_json"`
}

// CampaignResults is a struct representing the results from a campaign
type CampaignResults struct {
	Id      int64    `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Total   int64    `json:"total" gorm:"-"`
	Results []Result `json:"results,omitempty" gorm:"-"`
	Events  []Event  `json:"timeline,omitempty" gorm:"-"`
	SMTPs   []SMTP   `json:"smtps,omitempty" gorm:"-"`
}

// CampaignSummaries is a struct representing the overview of campaigns
type CampaignSummaries struct {
	Total     int64             `json:"total"`
	Campaigns []CampaignSummary `json:"campaigns"`
}

// CampaignSummary is a struct representing the overview of a single camaign
type CampaignSummary struct {
	Id            int64         `json:"id"`
	CreatedDate   time.Time     `json:"created_date"`
	LaunchDate    time.Time     `json:"launch_date"`
	SendByDate    time.Time     `json:"send_by_date"`
	CompletedDate time.Time     `json:"completed_date"`
	Status        string        `json:"status"`
	Name          string        `json:"name"`
	SourceType    string        `json:"source_type"`
	Stats         CampaignStats `json:"stats" gorm:"-"`
}

// CampaignStats is a struct representing the statistics for a single campaign
type CampaignStats struct {
	Total         int64 `json:"total"`
	EmailsSent    int64 `json:"sent"`
	OpenedEmail   int64 `json:"opened"`
	ClickedLink   int64 `json:"clicked"`
	SubmittedData int64 `json:"submitted_data"`
	EmailReported int64 `json:"email_reported"`
	ReportCount   int64 `json:"report_count"` // records in reports_ext (client/page)
	Error         int64 `json:"error"`
}

// Event contains the fields for an event
// that occurs during the campaign
type Event struct {
	Id         int64     `json:"-"`
	CampaignId int64     `json:"campaign_id"`
	Email      string    `json:"email"`
	Time       time.Time `json:"time"`
	Message    string    `json:"message"`
	Details    string    `json:"details"`
}

// EventDetails is a struct that wraps common attributes we want to store
// in an event
type EventDetails struct {
	Payload url.Values        `json:"payload"`
	Browser map[string]string `json:"browser"`
}

// EventError is a struct that wraps an error that occurs when sending an
// email to a recipient
type EventError struct {
	Error string `json:"error"`
}

// ErrCampaignNameNotSpecified indicates there was no template given by the user
var ErrCampaignNameNotSpecified = errors.New("Campaign name not specified")

// ErrGroupNotSpecified indicates there was no template given by the user
var ErrGroupNotSpecified = errors.New("No groups specified")

// ErrTemplateNotSpecified indicates there was no template given by the user
var ErrTemplateNotSpecified = errors.New("No email template specified")

// ErrPageNotSpecified indicates a landing page was not provided for the campaign
var ErrPageNotSpecified = errors.New("No landing page specified")

// ErrSMTPNotSpecified indicates a sending profile was not provided for the campaign
var ErrSMTPNotSpecified = errors.New("No sending profile specified")

// ErrTemplateNotFound indicates the template specified does not exist in the database
var ErrTemplateNotFound = errors.New("Template not found")

// ErrGroupNotFound indicates a group specified by the user does not exist in the database
var ErrGroupNotFound = errors.New("Group not found")

// ErrPageNotFound indicates a page specified by the user does not exist in the database
var ErrPageNotFound = errors.New("Page not found")

// ErrSMTPNotFound indicates a sending profile specified by the user does not exist in the database
var ErrSMTPNotFound = errors.New("Sending profile not found")

// ErrInvalidSendByDate indicates that the user specified a send by date that occurs before the
// launch date
var ErrInvalidSendByDate = errors.New("The launch date must be before the \"send emails by\" date")

// RecipientParameter is the URL parameter that points to the result ID for a recipient.
const RecipientParameter = "rid"

// Validate checks to make sure there are no invalid fields in a submitted campaign
func (c *Campaign) Validate() error {
	if c.SourceType == "" {
		c.SourceType = SourceTypeEmail
	}
	// Non-email activities (client / fixed page) don't require an email
	// template, target groups or sending profiles.
	if c.SourceType == SourceTypeClient || c.SourceType == SourceTypePage {
		if c.Name == "" {
			return ErrCampaignNameNotSpecified
		}
		return nil
	}
	switch {
	case c.Name == "":
		return ErrCampaignNameNotSpecified
	case len(c.Groups) == 0:
		return ErrGroupNotSpecified
	case c.Template.Name == "":
		return ErrTemplateNotSpecified
	case c.Page.Name == "":
		return ErrPageNotSpecified
	case c.SMTP.Name == "" && len(c.SMTPs) == 0:
		return ErrSMTPNotSpecified
	case !c.SendByDate.IsZero() && !c.LaunchDate.IsZero() && c.SendByDate.Before(c.LaunchDate):
		return ErrInvalidSendByDate
	}
	return nil
}

// UpdateStatus changes the campaign status appropriately
func (c *Campaign) UpdateStatus(s string) error {
	// This could be made simpler, but I think there's a bug in gorm
	return db.Table("campaigns").Where("id=?", c.Id).Update("status", s).Error
}

// GetCampaignForContext returns a Campaign containing only the fields needed
// to handle phishing server requests (tracking, reporting and landing page
// rendering): status, page id, user id, the base URL and the primary SMTP
// "From" address. It deliberately avoids the heavy getDetails() query (which
// loads results, events, groups, targets, attachments and stats) that would
// otherwise run for every tracking or landing page request.
func GetCampaignForContext(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := readDB().Table("campaigns").
		Select("id, user_id, page_id, status, url, smtp_id, source_type").
		Where("id=? AND user_id=?", id, uid).First(&c).Error
	if err != nil {
		return c, err
	}
	// Load the primary SMTP so that getFromAddress() returns the configured
	// "From" address used when rendering the landing page. This mirrors the
	// behavior of GetCampaign's getDetails().
	err = readDB().Table("smtp").Where("id=?", c.SMTPId).First(&c.SMTP).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return c, err
		}
		c.SMTP = SMTP{Name: "[Deleted]"}
		log.Warnf("%s: sending profile not found for campaign", err)
	}
	return c, nil
}

// AddEvent creates a new campaign event in the database
func AddEvent(e *Event, campaignID int64) error {
	e.CampaignId = campaignID
	e.Time = time.Now().UTC()

	whs, err := GetActiveWebhooks()
	if err == nil {
		whEndPoints := []webhook.EndPoint{}
		for _, wh := range whs {
			whEndPoints = append(whEndPoints, webhook.EndPoint{
				URL:    wh.URL,
				Secret: wh.Secret,
			})
		}
		webhook.SendAll(whEndPoints, e)
	} else {
		log.Errorf("error getting active webhooks: %v", err)
	}

	return db.Save(e).Error
}

// getDetails retrieves the related attributes of the campaign
// from the database. If the Events and the Results are not available,
// an error is returned. Otherwise, the attribute name is set to [Deleted],
// indicating the user deleted the attribute (template, smtp, etc.)
func (c *Campaign) getDetails() error {
	// Use explicit queries instead of Related() which was removed in GORM v2
	err := readDB().Where("campaign_id=?", c.Id).Find(&c.Results).Error
	if err != nil {
		log.Warnf("%s: results not found for campaign", err)
		return err
	}
	err = readDB().Where("campaign_id=?", c.Id).Find(&c.Events).Error
	if err != nil {
		log.Warnf("%s: events not found for campaign", err)
		return err
	}
	err = readDB().Table("templates").Select("id, name, envelope_sender, subject, modified_date").Where("id=?", c.TemplateId).First(&c.Template).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Template = Template{Name: "[Deleted]"}
		log.Warnf("%s: template not found for campaign", err)
	}
	err = readDB().Select("id, template_id, type, name").Where("template_id=?", c.Template.Id).Find(&c.Template.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}
	err = readDB().Table("pages").Select("id, user_id, name, html, capture_credentials, capture_passwords, redirect_url, modified_date").Where("id=?", c.PageId).First(&c.Page).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Page = Page{Name: "[Deleted]"}
		log.Warnf("%s: page not found for campaign", err)
	}
	err = readDB().Table("smtp").Where("id=?", c.SMTPId).First(&c.SMTP).Error
	if err != nil {
		// Check if the SMTP was deleted
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.SMTP = SMTP{Name: "[Deleted]"}
		log.Warnf("%s: sending profile not found for campaign", err)
	}
	err = readDB().Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}

	// Load multiple SMTPs from campaign_smtps join table
	c.SMTPs, err = GetCampaignSMTPRecords(c.Id, c.UserId)
	if err != nil {
		log.Warn(err)
	}

	// Load campaign stats
	c.Stats, err = getCampaignStats(c.Id, c.SourceType)
	if err != nil {
		log.Warnf("%s: stats not found for campaign", err)
	}

	// Load the report config for client/page type activities.
	if c.SourceType == SourceTypeClient || c.SourceType == SourceTypePage {
		rc, err := GetCampaignReportConfig(c)
		if err != nil {
			log.Warn(err)
		} else {
			c.ReportConfig = rc
		}
	}

	return nil
}

// getBaseURL returns the Campaign's configured URL.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getBaseURL() string {
	return c.URL
}

// getFromAddress returns the Campaign's configured SMTP "From" address.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getFromAddress() string {
	return c.SMTP.FromAddress
}

// generateSendDate creates a sendDate
func (c *Campaign) generateSendDate(idx int, totalRecipients int) time.Time {
	// If no send date is specified, just return the launch date
	if c.SendByDate.IsZero() || c.SendByDate.Equal(c.LaunchDate) {
		return c.LaunchDate
	}
	// Otherwise, we can calculate the range of minutes to send emails
	// (since we only poll once per minute)
	totalMinutes := c.SendByDate.Sub(c.LaunchDate).Minutes()

	// Next, we can determine how many minutes should elapse between emails
	minutesPerEmail := totalMinutes / float64(totalRecipients)

	// Then, we can calculate the offset for this particular email
	offset := int(minutesPerEmail * float64(idx))

	// Finally, we can just add this offset to the launch date to determine
	// when the email should be sent
	return c.LaunchDate.Add(time.Duration(offset) * time.Minute)
}

// getCampaignStats returns a CampaignStats object for the campaign with the
// given campaign ID. It also backfills numbers as appropriate with a running
// total, so that the values are aggregated.
//
// The results-based counters are computed in a single conditional-aggregate
// query; report/click stats are only queried for client/page campaigns.
func getCampaignStats(cid int64, sourceType string) (CampaignStats, error) {
	s := CampaignStats{}
	type aggRow struct {
		Total    int64
		Sent     int64
		Opened   int64
		Clicked  int64
		Sub      int64
		Err      int64
		Reported int64
	}
	var row aggRow
	err := readDB().Table("results").
		Select(`
			COUNT(*)                                                      AS total,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS sent,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS opened,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS clicked,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS sub,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS err,
			SUM(CASE WHEN reported = 1 THEN 1 ELSE 0 END)                AS reported
		`, EventSent, EventOpened, EventClicked, EventDataSubmit, Error).
		Where("campaign_id = ?", cid).
		Scan(&row).Error
	if err != nil {
		return s, err
	}
	// Apply running total backfill.
	s.Total = row.Total
	s.SubmittedData = row.Sub
	s.ClickedLink = row.Clicked + row.Sub
	s.OpenedEmail = row.Opened + row.Clicked + row.Sub
	s.EmailsSent = row.Sent + row.Opened + row.Clicked + row.Sub
	s.EmailReported = row.Reported
	s.Error = row.Err

	// Report and page-click stats only exist for client/page campaigns.
	// Email campaigns never write to reports_ext, so skip these queries.
	if sourceType == SourceTypeClient || sourceType == SourceTypePage {
		if err = readDB().Table("reports_ext").Where("campaign_id = ?", cid).Count(&s.ReportCount).Error; err != nil {
			return s, err
		}
		// Also populate SubmittedData for client/page campaigns (the
		// email-specific SubmittedData count from results is always zero for
		// these types).
		if s.ReportCount > 0 && s.SubmittedData == 0 {
			s.SubmittedData = s.ReportCount
			// page_click_stats unique index (campaign_id, vid) = total unique visitors.
			var totalVisitors int64
			if err = readDB().Table("page_click_stats").Where("campaign_id = ?", cid).Count(&totalVisitors).Error; err != nil {
				return s, err
			}
			s.OpenedEmail = totalVisitors
		}
	}

	// For page-type campaigns, add page click stats to ClickedLink.
	// The page_click_stats table records the total number of page opens
	// per visitor, flushed from memory periodically.
	if sourceType == SourceTypePage {
		var pageClicks int64
		if err = readDB().Table("page_click_stats").Where("campaign_id = ?", cid).
			Select("COALESCE(SUM(click_count), 0)").Scan(&pageClicks).Error; err != nil {
			return s, err
		}
		s.ClickedLink += pageClicks
	}

	return s, nil
}

// GetCampaigns returns a paginated page of campaigns owned by the given user.
func GetCampaigns(uid int64, pp PageParams) ([]Campaign, int64, error) {
	cs := []Campaign{}
	var total int64
	if err := readDB().Table("campaigns").Where("user_id=?", uid).Count(&total).Error; err != nil {
		log.Error(err)
		return cs, 0, err
	}
	query := readDB().Table("campaigns").Where("user_id=?", uid).Order("created_date DESC")
	query = query.Limit(pp.PageSize).Offset(pp.Offset())
	err := query.Find(&cs).Error
	if err != nil {
		log.Error(err)
		return cs, 0, err
	}
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, total, nil
}

// GetCampaignSummaries gets a paginated page of summary objects for the
// campaigns owned by the current user.
func GetCampaignSummaries(uid int64, pp PageParams) (CampaignSummaries, error) {
	overview := CampaignSummaries{}
	cs := []CampaignSummary{}
	var total int64
	if err := readDB().Table("campaigns").Where("user_id = ?", uid).Count(&total).Error; err != nil {
		log.Error(err)
		return overview, err
	}
	overview.Total = total
	// Get the basic campaign information
	query := readDB().Table("campaigns").Where("user_id = ?", uid)
	query = query.Select("id, name, created_date, launch_date, send_by_date, completed_date, status, source_type")
	query = query.Order("created_date DESC")
	query = query.Limit(pp.PageSize).Offset(pp.Offset())
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return overview, err
	}
	for i := range cs {
		s, err := getCampaignStats(cs[i].Id, cs[i].SourceType)
		if err != nil {
			log.Error(err)
			return overview, err
		}
		cs[i].Stats = s
	}
	overview.Campaigns = cs
	return overview, nil
}

// DashboardTimelineEntry holds per-campaign stats for the timeline chart.
type DashboardTimelineEntry struct {
	Id            int64     `json:"id"`
	Name          string    `json:"name"`
	CreatedDate   time.Time `json:"created_date"`
	Status        string    `json:"status"`
	Total         int64     `json:"total"`
	EmailsSent    int64     `json:"sent"`
	OpenedEmail   int64     `json:"opened"`
	ClickedLink   int64     `json:"clicked"`
	SubmittedData int64     `json:"submitted_data"`
	EmailReported int64     `json:"email_reported"`
	ReportCount   int64     `json:"report_count"`
	Error         int64     `json:"error"`
}

// DashboardStatsResponse is the lightweight response for dashboard charts.
type DashboardStatsResponse struct {
	Stats    CampaignStats            `json:"stats"`
	Timeline []DashboardTimelineEntry `json:"timeline"`
}

// GetDashboardStats returns aggregated campaign statistics using a single
// efficient SQL query, avoiding the N+1 query problem of GetCampaignSummaries.
func GetDashboardStats(uid int64) (DashboardStatsResponse, error) {
	resp := DashboardStatsResponse{}

	// Query 1: Aggregated stats across all campaigns for this user
	type aggRow struct {
		Total         int64
		Sent          int64
		Opened        int64
		Clicked       int64
		SubmittedData int64
		Reported      int64
		Error         int64
	}
	var row aggRow
	err := readDB().Table("results").
		Select(`
			COUNT(*)                                                      AS total,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS sent,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS opened,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS clicked,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS submitted_data,
			SUM(CASE WHEN reported = 1 THEN 1 ELSE 0 END)                AS reported,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)                  AS error
		`, EventSent, EventOpened, EventClicked, EventDataSubmit, Error).
		Where("campaign_id IN (SELECT id FROM campaigns WHERE user_id = ?)", uid).
		Scan(&row).Error
	if err != nil {
		return resp, err
	}

	// Apply running total backfill (same logic as getCampaignStats)
	s := CampaignStats{
		Total:         row.Total,
		SubmittedData: row.SubmittedData,
		ClickedLink:   row.Clicked + row.SubmittedData,
		OpenedEmail:   row.Opened + row.Clicked + row.SubmittedData,
		EmailsSent:    row.Sent + row.Opened + row.Clicked + row.SubmittedData,
		EmailReported: row.Reported,
		Error:         row.Error,
	}

	// Query reports_ext aggregated count
	if err = readDB().Table("reports_ext").
		Where("campaign_id IN (SELECT id FROM campaigns WHERE user_id = ?)", uid).
		Count(&s.ReportCount).Error; err != nil {
		return resp, err
	}
	if s.ReportCount > 0 && s.SubmittedData == 0 {
		s.SubmittedData = s.ReportCount
		// page_click_stats unique index (campaign_id, vid) = total unique visitors.
		var totalVisitors int64
		if err = readDB().Table("page_click_stats").
			Where("campaign_id IN (SELECT id FROM campaigns WHERE user_id = ?)", uid).
			Count(&totalVisitors).Error; err != nil {
			return resp, err
		}
		s.OpenedEmail = totalVisitors
	}

	// Add page click stats to ClickedLink for page-type campaigns.
	var pageClicks int64
	if err = readDB().Table("page_click_stats").
		Where("campaign_id IN (SELECT id FROM campaigns WHERE user_id = ? AND source_type = ?)", uid, SourceTypePage).
		Select("COALESCE(SUM(click_count), 0)").
		Scan(&pageClicks).Error; err != nil {
		return resp, err
	}
	s.ClickedLink += pageClicks

	resp.Stats = s

	// Query 2: Per-campaign timeline data (one row per campaign)
	type timelineRow struct {
		Id            int64
		Name          string
		CreatedDate   time.Time
		Status        string
		Total         int64
		Sent          int64
		Opened        int64
		Clicked       int64
		SubmittedData int64
		Reported      int64
		Error         int64
	}
	var rows []timelineRow
	err = readDB().Table("campaigns c").
		Select(`
			c.id, c.name, c.created_date, c.status,
			COUNT(r.id)                                                      AS total,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END)                   AS sent,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END)                   AS opened,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END)                   AS clicked,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END)                   AS submitted_data,
			SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END)                 AS reported,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END)                   AS error
		`, EventSent, EventOpened, EventClicked, EventDataSubmit, Error).
		Joins("LEFT JOIN results r ON r.campaign_id = c.id").
		Where("c.user_id = ?", uid).
		Group("c.id, c.name, c.created_date, c.status").
		Order("c.created_date DESC").
		Scan(&rows).Error
	if err != nil {
		return resp, err
	}

	// Batch per-campaign report / click stats so the timeline does not issue
	// N+1 queries on the loop below.
	type keyedCountRow struct {
		CampaignId int64
		Cnt        int64
	}
	reportCounts := map[int64]int64{}
	visitorCounts := map[int64]int64{}
	campaignClicks := map[int64]int64{}
	if len(rows) > 0 {
		ids := make([]int64, 0, len(rows))
		for _, r_ := range rows {
			ids = append(ids, r_.Id)
		}
		var repRows []keyedCountRow
		if err := readDB().Table("reports_ext").Select("campaign_id, COUNT(*) AS cnt").
			Where("campaign_id IN ?", ids).Group("campaign_id").Scan(&repRows).Error; err != nil {
			return resp, err
		}
		for _, r_ := range repRows {
			reportCounts[r_.CampaignId] = r_.Cnt
		}
		var visRows []keyedCountRow
		if err := readDB().Table("page_click_stats").Select("campaign_id, COUNT(*) AS cnt").
			Where("campaign_id IN ?", ids).Group("campaign_id").Scan(&visRows).Error; err != nil {
			return resp, err
		}
		for _, r_ := range visRows {
			visitorCounts[r_.CampaignId] = r_.Cnt
		}
		type keyedSumRow struct {
			CampaignId int64
			Sum        int64
		}
		var sumRows []keyedSumRow
		if err := readDB().Table("page_click_stats").
			Select("campaign_id, COALESCE(SUM(click_count), 0) AS sum").
			Where("campaign_id IN ?", ids).Group("campaign_id").Scan(&sumRows).Error; err != nil {
			return resp, err
		}
		for _, r_ := range sumRows {
			campaignClicks[r_.CampaignId] = r_.Sum
		}
	}

	// Convert timelineRows to DashboardTimelineEntry with running total backfill
	timeline := make([]DashboardTimelineEntry, 0, len(rows))
	for _, r_ := range rows {
		entry := DashboardTimelineEntry{
			Id:            r_.Id,
			Name:          r_.Name,
			CreatedDate:   r_.CreatedDate,
			Status:        r_.Status,
			Total:         r_.Total,
			SubmittedData: r_.SubmittedData,
			ClickedLink:   r_.Clicked + r_.SubmittedData,
			OpenedEmail:   r_.Opened + r_.Clicked + r_.SubmittedData,
			EmailsSent:    r_.Sent + r_.Opened + r_.Clicked + r_.SubmittedData,
			EmailReported: r_.Reported,
			Error:         r_.Error,
		}
		entry.ReportCount = reportCounts[r_.Id]
		if entry.ReportCount > 0 && entry.SubmittedData == 0 {
			entry.SubmittedData = entry.ReportCount
			// page_click_stats unique index (campaign_id, vid) = total unique visitors.
			entry.OpenedEmail = visitorCounts[r_.Id]
		}
		// Add page click stats for this campaign.
		entry.ClickedLink += campaignClicks[r_.Id]
		timeline = append(timeline, entry)
	}
	resp.Timeline = timeline
	return resp, nil
}

// GetCampaignSummary gets the summary object for a campaign specified by the campaign ID
func GetCampaignSummary(id int64, uid int64) (CampaignSummary, error) {
	cs := CampaignSummary{}
	query := readDB().Table("campaigns").Where("user_id = ? AND id = ?", uid, id)
	query = query.Select("id, name, created_date, launch_date, send_by_date, completed_date, status, source_type")
	// GORM v2: Scan does not return ErrRecordNotFound for zero rows; use First instead
	err := query.First(&cs).Error
	if err != nil {
		log.Error(err)
		return cs, err
	}
	s, err := getCampaignStats(cs.Id, cs.SourceType)
	if err != nil {
		log.Error(err)
		return cs, err
	}
	cs.Stats = s
	return cs, nil
}

// GetCampaignMailContext returns a campaign object with just the relevant
// data needed to generate and send emails. This includes the top-level
// metadata, the template, and the sending profile.
//
// This should only ever be used if you specifically want this lightweight
// context, since it returns a non-standard campaign object.
// ref: #1726
func GetCampaignMailContext(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := readDB().Where("id = ?", id).Where("user_id = ?", uid).First(&c).Error
	if err != nil {
		return c, err
	}
	err = readDB().Table("smtp").Where("id=?", c.SMTPId).First(&c.SMTP).Error
	if err != nil {
		return c, err
	}
	err = readDB().Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return c, err
	}
	err = readDB().Table("templates").Where("id=?", c.TemplateId).First(&c.Template).Error
	if err != nil {
		return c, err
	}
	err = readDB().Where("template_id=?", c.Template.Id).Find(&c.Template.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return c, err
	}

	// Load multiple SMTPs from campaign_smtps join table
	c.SMTPs, err = GetCampaignSMTPRecords(c.Id, uid)
	if err != nil {
		log.Warn(err)
	}
	return c, nil
}

// GetCampaign returns the campaign, if it exists, specified by the given id and user_id.
func GetCampaign(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := readDB().Where("id = ?", id).Where("user_id = ?", uid).First(&c).Error
	if err != nil {
		log.Errorf("%s: campaign not found", err)
		return c, err
	}
	err = c.getDetails()
	return c, err
}

// GetCampaignResults returns a paginated page of campaign results for the
// given campaign.
func GetCampaignResults(id int64, uid int64, pp PageParams) (CampaignResults, error) {
	cr := CampaignResults{}
	err := readDB().Table("campaigns").Where("id=? and user_id=?", id, uid).First(&cr).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"campaign_id": id,
			"error":       err,
		}).Error(err)
		return cr, err
	}
	if err := readDB().Table("results").Where("campaign_id=? and user_id=?", cr.Id, uid).Count(&cr.Total).Error; err != nil {
		log.Errorf("%s: results not found for campaign", err)
		return cr, err
	}
	query := readDB().Table("results").Where("campaign_id=? and user_id=?", cr.Id, uid).Order("id ASC")
	if pp.Valid() {
		query = query.Limit(pp.PageSize).Offset(pp.Offset())
	}
	err = query.Find(&cr.Results).Error
	if err != nil {
		log.Errorf("%s: results not found for campaign", err)
		return cr, err
	}
	err = readDB().Table("events").Where("campaign_id=?", cr.Id).Find(&cr.Events).Error
	if err != nil {
		log.Errorf("%s: events not found for campaign", err)
		return cr, err
	}
	// Load the SMTP sending profiles associated with this campaign
	cr.SMTPs, err = GetCampaignSMTPRecords(cr.Id, uid)
	if err != nil {
		log.Warnf("%s: smtps not found for campaign", err)
	}
	// Populate smtp_from_address for each result
	smtpMap := make(map[int64]string)
	for _, s := range cr.SMTPs {
		smtpMap[s.Id] = s.FromAddress
	}
	for i := range cr.Results {
		if addr, ok := smtpMap[cr.Results[i].SMTPId]; ok {
			cr.Results[i].SMTPFromAddress = addr
		}
	}
	return cr, err
}

// GetQueuedCampaigns returns the campaigns that are queued up for this given minute
func GetQueuedCampaigns(t time.Time) ([]Campaign, error) {
	cs := []Campaign{}
	err := readDB().Where("launch_date <= ?", t).
		Where("status = ?", CampaignQueued).Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	log.Infof("Found %d Campaigns to run\n", len(cs))
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// PostCampaign inserts a campaign and all associated records into the database.
func PostCampaign(c *Campaign, uid int64) error {
	err := c.Validate()
	if err != nil {
		return err
	}
	if c.SourceType == SourceTypeClient || c.SourceType == SourceTypePage {
		return postReportCampaign(c, uid)
	}
	// Fill in the details
	c.UserId = uid
	c.CreatedDate = time.Now().UTC()
	c.CompletedDate = time.Time{}
	c.Status = CampaignQueued
	if c.LaunchDate.IsZero() {
		c.LaunchDate = c.CreatedDate
	} else {
		c.LaunchDate = c.LaunchDate.UTC()
	}
	if !c.SendByDate.IsZero() {
		c.SendByDate = c.SendByDate.UTC()
	}
	if c.LaunchDate.Before(c.CreatedDate) || c.LaunchDate.Equal(c.CreatedDate) {
		c.Status = CampaignInProgress
	}
	// Check to make sure all the groups already exist.
	for i, g := range c.Groups {
		c.Groups[i], err = GetGroupByName(g.Name, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"group": g.Name,
			}).Error("Group does not exist")
			return ErrGroupNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
	}
	// Check to make sure the template exists
	t, err := GetTemplateByName(c.Template.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"template": c.Template.Name,
		}).Error("Template does not exist")
		return ErrTemplateNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	c.Template = t
	c.TemplateId = t.Id
	// Check to make sure the page exists
	p, err := GetPageByName(c.Page.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"page": c.Page.Name,
		}).Error("Page does not exist")
		return ErrPageNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	c.Page = p
	c.PageId = p.Id
	// Check to make sure the sending profile(s) exist.
	// Support both the legacy single SMTP (c.SMTP.Name) and the new
	// multi-SMTP (c.SMTPs) payload.  When SMTPs is provided it takes
	// precedence; otherwise we fall back to the single SMTP.
	var resolvedSMTPs []SMTP
	if len(c.SMTPs) > 0 {
		seen := make(map[string]bool)
		for _, sReq := range c.SMTPs {
			if seen[sReq.Name] {
				continue // deduplicate
			}
			seen[sReq.Name] = true
			s, err := GetSMTPByName(sReq.Name, uid)
			if err == gorm.ErrRecordNotFound {
				log.WithFields(logrus.Fields{"smtp": sReq.Name}).Error("Sending profile does not exist")
				return ErrSMTPNotFound
			} else if err != nil {
				log.Error(err)
				return err
			}
			resolvedSMTPs = append(resolvedSMTPs, s)
		}
		// For backward-compat, also set the primary SMTP fields
		c.SMTP = resolvedSMTPs[0]
		c.SMTPId = resolvedSMTPs[0].Id
	} else {
		s, err := GetSMTPByName(c.SMTP.Name, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{"smtp": c.SMTP.Name}).Error("Sending profile does not exist")
			return ErrSMTPNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		c.SMTP = s
		c.SMTPId = s.Id
		resolvedSMTPs = []SMTP{s}
	}
	// Insert into the DB
	err = db.Save(c).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Save campaign_smtps join table records
	smtpIds := make([]int64, len(resolvedSMTPs))
	for i, s := range resolvedSMTPs {
		smtpIds[i] = s.Id
	}
	if err := PostCampaignSMTPs(db, c.Id, smtpIds); err != nil {
		log.Error(err)
		return err
	}
	c.SMTPs = resolvedSMTPs
	err = AddEvent(&Event{Message: "Campaign Created"}, c.Id)
	if err != nil {
		log.Error(err)
	}
	// Insert all the results
	// Step 1: merge all group targets and deduplicate, preserving order.
	var uniqueTargets []Target
	seen := make(map[string]bool)
	for _, g := range c.Groups {
		for _, t := range g.Targets {
			if seen[t.Email] {
				continue
			}
			seen[t.Email] = true
			uniqueTargets = append(uniqueTargets, t)
		}
	}

	realTotal := len(uniqueTargets)
	numSMTPs := len(resolvedSMTPs)
	basePerSMTP := realTotal / numSMTPs
	remainder := realTotal % numSMTPs

	// Step 2: build the results and maillogs for every recipient, then insert
	// them in bulk instead of one round-trip per recipient.
	// First `remainder` profiles each get (basePerSMTP+1) recipients,
	// the remaining profiles each get basePerSMTP recipients.
	results := make([]Result, 0, realTotal)
	maillogs := make([]MailLog, 0, realTotal)
	for recipientIndex, t := range uniqueTargets {
		sendDate := c.generateSendDate(recipientIndex, realTotal)
		var smtpIndex int
		if basePerSMTP == 0 {
			// Fewer recipients than SMTPs, fall back to round-robin
			smtpIndex = recipientIndex % numSMTPs
		} else if recipientIndex < remainder*(basePerSMTP+1) {
			smtpIndex = recipientIndex / (basePerSMTP + 1)
		} else {
			smtpIndex = remainder + (recipientIndex-remainder*(basePerSMTP+1))/basePerSMTP
		}
		assignedSMTP := resolvedSMTPs[smtpIndex]
		r := Result{
			BaseRecipient: BaseRecipient{
				Email:    t.Email,
				Position: t.Position,
				FullName: t.FullName,
			},
			SMTPId:       assignedSMTP.Id,
			Status:       StatusScheduled,
			CampaignId:   c.Id,
			UserId:       c.UserId,
			SendDate:     sendDate,
			Reported:     false,
			ModifiedDate: c.CreatedDate,
		}
		processing := false
		if r.SendDate.Before(c.CreatedDate) || r.SendDate.Equal(c.CreatedDate) {
			r.Status = StatusSending
			processing = true
		}
		results = append(results, r)
		maillogs = append(maillogs, MailLog{
			UserId:     c.UserId,
			CampaignId: c.Id,
			SendDate:   sendDate,
			Processing: processing,
		})
	}
	tx := db.Begin()
	// Generate unique result ids for the whole batch up front.
	rIds, err := generateUniqueResultIds(len(results), tx)
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	for i := range results {
		results[i].RId = rIds[i]
		maillogs[i].RId = rIds[i]
	}
	c.Results = results
	if err = insertResultsBulk(tx, results); err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	if err = insertMailLogsBulk(tx, maillogs); err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// postReportCampaign persists a client/page type activity. It generates a
// per-campaign random report salt (persisted so keys survive restarts),
// normalizes the dynamic report configuration and stores the campaign without
// any email-related setup.
func postReportCampaign(c *Campaign, uid int64) error {
	c.UserId = uid
	c.CreatedDate = time.Now().UTC()
	c.CompletedDate = time.Time{}
	c.Status = CampaignInProgress
	// Generate a per-campaign random salt used to derive the report key.
	salt, err := GenerateReportSalt()
	if err != nil {
		log.Error(err)
		return err
	}
	c.ReportSalt = salt
	// Page-type campaigns need a valid landing page. Resolve the page by
	// name so that PageId is populated and renderFixedPage can serve it.
	if c.SourceType == SourceTypePage {
		if c.Page.Name == "" {
			return ErrPageNotSpecified
		}
		p, err := GetPageByName(c.Page.Name, uid)
		if err == gorm.ErrRecordNotFound {
			return ErrPageNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		c.Page = p
		c.PageId = p.Id
	}
	// Normalize the report configuration.
	rc := c.ReportConfig
	if rc == nil {
		rc, err = GetCampaignReportConfig(c)
		if err != nil {
			return err
		}
	}
	if rc == nil || len(rc.Fields) == 0 {
		if c.SourceType == SourceTypePage {
			rc = NewPageReportConfig()
		} else {
			rc = NewReportConfig()
		}
	}
	c.ReportConfig = rc
	c.ReportConfigJSON = rc.Marshal()
	if err := db.Save(c).Error; err != nil {
		log.Error(err)
		return err
	}
	if err := AddEvent(&Event{Message: "Campaign Created"}, c.Id); err != nil {
		log.Error(err)
	}
	return nil
}

// generateUniqueResultIds generates n unique result ids, checking for
// collisions both against the database and within the generated batch using a
// single batched query per round instead of one query per id.
func generateUniqueResultIds(n int, tx *gorm.DB) ([]string, error) {
	ids := make([]string, n)
	for i := range ids {
		id, err := generateResultId()
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	for attempt := 0; attempt < 8; attempt++ {
		// Ids that already exist in the database.
		blocked := make(map[string]bool, n)
		for _, chunk := range chunkStrings(ids) {
			var existing []string
			if err := tx.Table("results").Where("r_id IN (?)", chunk).Pluck("r_id", &existing).Error; err != nil {
				return nil, err
			}
			for _, e := range existing {
				blocked[e] = true
			}
		}
		// Duplicates within the generated batch.
		seen := make(map[string]bool, n)
		for _, id := range ids {
			if seen[id] {
				blocked[id] = true
			}
			seen[id] = true
		}
		needRegen := false
		for i := range ids {
			if !blocked[ids[i]] {
				continue
			}
			needRegen = true
			for {
				id, err := generateResultId()
				if err != nil {
					return nil, err
				}
				if !seen[id] && !blocked[id] {
					ids[i] = id
					seen[id] = true
					break
				}
			}
		}
		if !needRegen {
			return ids, nil
		}
	}
	return nil, errors.New("unable to generate unique result ids after repeated attempts")
}

// insertResultsBulk inserts the results in chunks, keeping the number of bind
// variables within SQLite's per-statement limit.
func insertResultsBulk(tx *gorm.DB, results []Result) error {
	if len(results) == 0 {
		return nil
	}
	const batchSize = 50
	for start := 0; start < len(results); start += batchSize {
		end := start + batchSize
		if end > len(results) {
			end = len(results)
		}
		sql := "INSERT INTO results (campaign_id, user_id, r_id, email, full_name, position, status, ip, latitude, longitude, send_date, reported, modified_date, smtp_id) VALUES"
		var args []interface{}
		for i := start; i < end; i++ {
			if i > start {
				sql += ","
			}
			sql += " (?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
			r := results[i]
			args = append(args, r.CampaignId, r.UserId, r.RId, r.Email, r.FullName, r.Position, r.Status, r.IP, r.Latitude, r.Longitude, r.SendDate, r.Reported, r.ModifiedDate, r.SMTPId)
		}
		if err := tx.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// insertMailLogsBulk inserts the maillogs in chunks.
func insertMailLogsBulk(tx *gorm.DB, maillogs []MailLog) error {
	if len(maillogs) == 0 {
		return nil
	}
	const batchSize = 100
	for start := 0; start < len(maillogs); start += batchSize {
		end := start + batchSize
		if end > len(maillogs) {
			end = len(maillogs)
		}
		sql := "INSERT INTO mail_logs (campaign_id, user_id, send_date, send_attempt, r_id, processing) VALUES"
		var args []interface{}
		for i := start; i < end; i++ {
			if i > start {
				sql += ","
			}
			sql += " (?,?,?,?,?,?)"
			m := maillogs[i]
			args = append(args, m.CampaignId, m.UserId, m.SendDate, m.SendAttempt, m.RId, m.Processing)
		}
		if err := tx.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// deleteAllCampaignsForUser deletes every campaign belonging to the given user.
// It fetches only campaign IDs (not full objects) and delegates the actual
// cleanup to DeleteCampaign so that results, events, maillogs, reports and
// campaign_smtps join records are properly removed.
func deleteAllCampaignsForUser(uid int64) error {
	var ids []int64
	if err := db.Table("campaigns").Where("user_id=?", uid).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := DeleteCampaign(id); err != nil {
			return err
		}
	}
	return nil
}

// DeleteCampaign deletes the specified campaign
func DeleteCampaign(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Deleting campaign")
	// Delete campaign_smtps join records
	err := DeleteCampaignSMTPsByCampaign(id)
	if err != nil {
		log.Error(err)
	}
	// Delete all the campaign results
	err = db.Where("campaign_id=?", id).Delete(&Result{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&Event{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = DeleteCampaignReports(id)
	if err != nil {
		log.Error(err)
		return err
	}
	// Delete the campaign
	err = db.Delete(&Campaign{Id: id}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// CompleteCampaign effectively "ends" a campaign.
// Any future emails clicked will return a simple "404" page.
func CompleteCampaign(id int64, uid int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Marking campaign as complete")
	c, err := GetCampaign(id, uid)
	if err != nil {
		return err
	}
	// Delete any maillogs still set to be sent out, preventing future emails
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Don't overwrite original completed time
	if c.Status == CampaignComplete {
		return nil
	}
	// Mark the campaign as complete
	c.CompletedDate = time.Now().UTC()
	c.Status = CampaignComplete
	err = db.Model(&Campaign{}).Where("id=? and user_id=?", id, uid).
		Select([]string{"completed_date", "status"}).UpdateColumns(&c).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// LaunchCampaign marks a scheduled/queued campaign as in_progress and sets its launch date to now.
func LaunchCampaign(id int64, uid int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Launching campaign")
	now := time.Now().UTC()
	err := db.Model(&Campaign{}).Where("id=? and user_id=?", id, uid).
		Where("status IN (?)", []string{CampaignScheduled, CampaignQueued}).
		Updates(map[string]interface{}{
			"status":      CampaignInProgress,
			"launch_date": now,
		}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
