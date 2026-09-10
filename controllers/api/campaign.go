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
	switch {
	case r.Method == "GET":
		pp := parsePagination(r)
		cs, total, err := models.GetCampaigns(ctx.Get(r, "user_id").(int64), pp)
		if err != nil {
			log.Error(err)
		}
		ListResponse(w, cs, total, http.StatusOK)
	//POST: Create a new campaign and return it as JSON
	case r.Method == "POST":
		c := models.Campaign{}
		// Put the request into a campaign
		err := json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		err = models.PostCampaign(&c, ctx.Get(r, "user_id").(int64))
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		// If the campaign is scheduled to launch immediately, send it to the worker.
		// Otherwise, the worker will pick it up at the scheduled time
		if c.Status == models.CampaignInProgress && c.SourceType != models.SourceTypeClient && c.SourceType != models.SourceTypePage {
			go as.worker.LaunchCampaign(c)
		}
		SuccessResponse(w, c, http.StatusCreated)
	}
}

// CampaignsSummary returns the summary for the current user's campaigns
func (as *Server) CampaignsSummary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		pp := parsePagination(r)
		cs, err := models.GetCampaignSummaries(ctx.Get(r, "user_id").(int64), pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ListResponse(w, cs.Campaigns, cs.Total, http.StatusOK)
	}
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
	switch {
	case r.Method == "GET":
		pp := parsePagination(r)
		cr, err := models.GetCampaignResults(id, uid, pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, "Campaign not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, cr, http.StatusOK)
	case r.Method == "DELETE":
		err := models.DeleteCampaign(id)
		if err != nil {
			ErrorResponse(w, "Error deleting campaign", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Campaign deleted successfully!", http.StatusOK)
	}
}

// CampaignResultsExport downloads the campaign results as a CSV file.
// For email campaigns, each row includes the recipient's fixed fields, event
// timestamps (sent / opened / clicked / data submitted / reported), plus any
// dynamic data captured from DataSubmit events (e.g. submitted credentials).
// This is the single comprehensive CSV export for email campaigns, combining
// what used to be separate "results" and "events" exports.
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
		// Collect the earliest timestamp for each event type
		var sentTime, openedTime, clickedTime, dataSubmitTime, reportedTime time.Time
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
				if dataSubmitTime.IsZero() || ev.Time.Before(dataSubmitTime) {
					dataSubmitTime = ev.Time
				}
			case models.EventReported:
				if reportedTime.IsZero() || ev.Time.Before(reportedTime) {
					reportedTime = ev.Time
				}
			}
		}
		// Helper: return nil for zero time so the CSV cell is empty
		zeroToNil := func(t time.Time) interface{} {
			if t.IsZero() {
				return nil
			}
			return t
		}
		row := util.CSVRow{
			Fixed: []interface{}{
				res.RId, res.Email, res.FullName, res.Position,
				res.Status, res.IP,
				res.SendDate, zeroToNil(sentTime), zeroToNil(openedTime),
				zeroToNil(clickedTime), zeroToNil(dataSubmitTime),
				zeroToNil(reportedTime),
				res.Reported, res.ModifiedDate, res.SMTPFromAddress,
			},
			Data: map[string]interface{}{},
		}
		// Extract detail data from ALL events, not just DataSubmit.
		// Every event carries a Details JSON with "browser" (address,
		// user-agent) and optionally "payload" (form data).  The frontend
		// timeline shows all of these, so the CSV must include them too.
		// Columns are prefixed with the event type to avoid collisions
		// (e.g. clicked_ip, opened_user_agent, data_submitted_password).
		for _, ev := range res.Events {
			if ev.Details == "" {
				continue
			}
			detailMap := map[string]interface{}{}
			if json.Unmarshal([]byte(ev.Details), &detailMap) != nil {
				continue
			}
			// Determine the column-name prefix for this event type
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
			// Extract browser fields (IP, user-agent, etc.)
			if browser, ok := detailMap["browser"]; ok {
				if bm, ok := browser.(map[string]interface{}); ok {
					for k, v := range bm {
						colName := prefix + "_" + k
						if _, exists := row.Data[colName]; !exists {
							row.Data[colName] = v
						}
					}
				}
			}
			// Extract payload fields (submitted form data) — typically
			// only present on DataSubmit events, but handle any.
			if payload, ok := detailMap["payload"]; ok {
				if pm, ok := payload.(map[string]interface{}); ok {
					for k, v := range pm {
						colName := prefix + "_" + k
						if _, exists := row.Data[colName]; !exists {
							if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
								row.Data[colName] = fmt.Sprintf("%v", arr[0])
							} else {
								row.Data[colName] = v
							}
						}
					}
				}
			}
		}
		rows = append(rows, row)
	}
	writeCSVFile(w, cr.Name, "results", fixedKeys, rows)
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
