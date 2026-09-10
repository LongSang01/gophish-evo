package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/util"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// Campaigns returns a list of campaigns if requested via GET.
// If requested via POST, APICampaigns creates a new campaign and returns a reference to it.
func (as *Server) Campaigns(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		cs, total, err := models.GetCampaigns(uid, pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ListResponse(w, cs, total, http.StatusOK)
	case http.MethodPost:
		c := models.Campaign{}
		err := json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		err = models.PostCampaign(&c, uid)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		if c.Status == models.CampaignInProgress && c.SourceType != models.SourceTypeClient && c.SourceType != models.SourceTypePage {
			go as.worker.LaunchCampaign(c)
		}
		SuccessResponse(w, c, http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CampaignsSummary returns the summary for the current user's campaigns
func (as *Server) CampaignsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pp := parsePagination(r)
	cs, err := models.GetCampaignSummaries(ctx.Get(r, "user_id").(int64), pp)
	if err != nil {
		log.Error(err)
		ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ListResponse(w, cs.Campaigns, cs.Total, http.StatusOK)
}

// DashboardStats returns lightweight aggregated stats for dashboard charts,
// avoiding the N+1 query problem of loading all campaign details.
func (as *Server) DashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp, err := models.GetDashboardStats(ctx.Get(r, "user_id").(int64))
	if err != nil {
		log.Error(err)
		ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	SuccessResponse(w, resp, http.StatusOK)
}

// Campaign returns details about the requested campaign. If the campaign is not
// valid, APICampaign returns null.
//
// GET returns campaign metadata and paginated results (merged from the former
// /campaigns/:id/results endpoint). The /campaigns/:id/summary endpoint can be
// used separately for stats and chart data.
func (as *Server) Campaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		cr, err := models.GetCampaignResults(id, uid, pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, "Campaign not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, cr, http.StatusOK)
	case http.MethodDelete:
		err := models.DeleteCampaign(id)
		if err != nil {
			ErrorResponse(w, "Error deleting campaign", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Campaign deleted successfully!", http.StatusOK)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CampaignResultsExport downloads the campaign results as a CSV file.
// For email campaigns, each row includes the recipient's fixed fields, event
// timestamps (sent / opened / clicked / data submitted / reported), plus any
// dynamic data captured from DataSubmit events (e.g. submitted credentials).
// This is the single comprehensive CSV export for email campaigns, combining
// what used to be separate "results" and "events" exports.
//
// When a recipient submits data multiple times (multiple DataSubmit events),
// a separate CSV row is generated for each submission so that every piece of
// captured data is included in the export.
func (as *Server) CampaignResultsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	cr, err := models.GetCampaignResults(id, ctx.Get(r, "user_id").(int64), models.PageParams{})
	if err != nil {
		log.Error(err)
		ErrorResponse(w, "Campaign not found", http.StatusNotFound)
		return
	}
	fixedKeys := []string{
		"id", "email", "full_name", "position", "status", "ip",
		"send_date", "sent_time", "opened_time", "clicked_time",
		"data_submitted_time", "reported_time",
		"reported", "modified_date", "smtp_from_address",
	}
	rows := make([]util.CSVRow, 0, len(cr.Results))
	for i := range cr.Results {
		res := &cr.Results[i]
		// Collect the earliest timestamp for each event type.
		var sentTime, openedTime, clickedTime, reportedTime time.Time
		// Collect all DataSubmit events so we can emit one row per submission.
		var dataSubmitEvents []models.Event
		// Shared dynamic data from non-DataSubmit events (browser info, etc.)
		sharedData := map[string]interface{}{}

		for _, ev := range res.Events {
			switch ev.Message {
			case models.EventSent:
				if sentTime.IsZero() || ev.Time.Before(sentTime) {
					sentTime = ev.Time
				}
			case models.EventOpened:
				if openedTime.IsZero() || ev.Time.Before(openedTime) {
					openedTime = ev.Time
				}
			case models.EventClicked:
				if clickedTime.IsZero() || ev.Time.Before(clickedTime) {
					clickedTime = ev.Time
				}
			case models.EventDataSubmit:
				dataSubmitEvents = append(dataSubmitEvents, ev)
			case models.EventReported:
				if reportedTime.IsZero() || ev.Time.Before(reportedTime) {
					reportedTime = ev.Time
				}
			}
			// Extract detail data from non-DataSubmit events as shared columns.
			if ev.Message != models.EventDataSubmit && ev.Details != "" {
				extractEventDetails(ev, sharedData)
			}
		}

		// Helper: return nil for zero time so the CSV cell is empty
		zeroToNil := func(t time.Time) interface{} {
			if t.IsZero() {
				return nil
			}
			return t
		}

		if len(dataSubmitEvents) == 0 {
			// No submissions — emit a single row with event timestamps only.
			row := util.CSVRow{
				Fixed: []interface{}{
					res.RId, res.Email, res.FullName, res.Position,
					res.Status, res.IP,
					res.SendDate, zeroToNil(sentTime), zeroToNil(openedTime),
					zeroToNil(clickedTime), nil,
					zeroToNil(reportedTime),
					res.Reported, res.ModifiedDate, res.SMTPFromAddress,
				},
				Data: sharedData,
			}
			rows = append(rows, row)
		} else {
			// One or more submissions — emit a row for each DataSubmit event.
			for _, dsev := range dataSubmitEvents {
				// Clone the shared data so each row is independent.
				rowData := map[string]interface{}{}
				for k, v := range sharedData {
					rowData[k] = v
				}
				// Extract payload & browser data from this specific submission.
				extractEventDetails(dsev, rowData)
				row := util.CSVRow{
					Fixed: []interface{}{
						res.RId, res.Email, res.FullName, res.Position,
						res.Status, res.IP,
						res.SendDate, zeroToNil(sentTime), zeroToNil(openedTime),
						zeroToNil(clickedTime), zeroToNil(dsev.Time),
						zeroToNil(reportedTime),
						res.Reported, res.ModifiedDate, res.SMTPFromAddress,
					},
					Data: rowData,
				}
				rows = append(rows, row)
			}
		}
	}
	writeCSVFile(w, cr.Name, "results", fixedKeys, rows)
}

// extractEventDetails parses an event's Details JSON and merges its browser
// and payload fields into the target map, using the event type as a column
// prefix (e.g. clicked_ip, data_submitted_password). Fields that already
// exist in the map are not overwritten.
func extractEventDetails(ev models.Event, target map[string]interface{}) {
	if ev.Details == "" {
		return
	}
	detailMap := map[string]interface{}{}
	if json.Unmarshal([]byte(ev.Details), &detailMap) != nil {
		return
	}
	prefix := ""
	switch ev.Message {
	case models.EventSent:
		prefix = "sent"
	case models.EventOpened:
		prefix = "opened"
	case models.EventClicked:
		prefix = "clicked"
	case models.EventDataSubmit:
		prefix = "data_submitted"
	case models.EventReported:
		prefix = "reported"
	default:
		prefix = ev.Message
	}
	if browser, ok := detailMap["browser"]; ok {
		if bm, ok := browser.(map[string]interface{}); ok {
			for k, v := range bm {
				colName := prefix + "_" + k
				if _, exists := target[colName]; !exists {
					target[colName] = v
				}
			}
		}
	}
	if payload, ok := detailMap["payload"]; ok {
		if pm, ok := payload.(map[string]interface{}); ok {
			for k, v := range pm {
				colName := prefix + "_" + k
				if _, exists := target[colName]; !exists {
					if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
						target[colName] = fmt.Sprintf("%v", arr[0])
					} else {
						target[colName] = v
					}
				}
			}
		}
	}
}

// jsonKeys returns the keys of a JSON-serialized struct in serialization order,
// matching Object.keys() on the parsed JSON object in the browser.
func jsonKeys(v interface{}) ([]string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("expected JSON object")
	}
	var keys []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, errors.New("expected JSON object key")
		}
		keys = append(keys, key)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// resultMap converts a Result into a map keyed by its JSON field names so that
// CSV columns can be filled positionally.
func resultMap(res *models.Result) (map[string]interface{}, error) {
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// CampaignEventsExport downloads the campaign timeline events as a CSV file.
func (as *Server) CampaignEventsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	cr, err := models.GetCampaignResults(id, ctx.Get(r, "user_id").(int64), models.PageParams{})
	if err != nil {
		log.Error(err)
		ErrorResponse(w, "Campaign not found", http.StatusNotFound)
		return
	}
	events := cr.AllEvents()
	fixedKeys := []string{"campaign_id", "email", "time", "message", "details"}
	rows := make([]util.CSVRow, 0, len(events))
	for i := range events {
		e := &events[i]
		rows = append(rows, util.CSVRow{Fixed: []interface{}{e.CampaignId, e.Email, e.Time, e.Message, e.Details}})
	}
	writeCSVFile(w, cr.Name, "events", fixedKeys, rows)
}

// CampaignSummary returns the summary for a given campaign.
func (as *Server) CampaignSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch {
	case r.Method == "GET":
		cs, err := models.GetCampaignSummary(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				ErrorResponse(w, "Campaign not found", http.StatusNotFound)
			} else {
				ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			}
			log.Error(err)
			return
		}
		SuccessResponse(w, cs, http.StatusOK)
	}
}

// CampaignComplete effectively "ends" a campaign.
// Future phishing emails clicked will return a simple "404" page.
func (as *Server) CampaignComplete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch {
	case r.Method == "GET":
		err := models.CompleteCampaign(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			ErrorResponse(w, "Error completing campaign", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Campaign completed successfully!", http.StatusOK)
	}
}

// CampaignLaunch launches a scheduled or queued campaign immediately.
func (as *Server) CampaignLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	c, err := models.GetCampaign(id, uid)
	if err != nil {
		ErrorResponse(w, "Campaign not found", http.StatusNotFound)
		return
	}
	if c.SourceType != models.SourceTypeEmail {
		ErrorResponse(w, "Only email campaigns can be launched", http.StatusBadRequest)
		return
	}
	if c.Status != models.CampaignScheduled && c.Status != models.CampaignQueued {
		ErrorResponse(w, "Campaign is not in a launchable state", http.StatusBadRequest)
		return
	}
	err = models.LaunchCampaign(id, uid)
	if err != nil {
		ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Launch via worker
	go as.worker.LaunchCampaign(c)
	ActionResponse(w, "Campaign launched successfully!", http.StatusOK)
}
