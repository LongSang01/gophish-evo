package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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
		pagedJSONResponse(w, http.StatusOK, pp, cs, total)
	//POST: Create a new campaign and return it as JSON
	case r.Method == "POST":
		c := models.Campaign{}
		// Put the request into a campaign
		err := json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}
		err = models.PostCampaign(&c, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		// If the campaign is scheduled to launch immediately, send it to the worker.
		// Otherwise, the worker will pick it up at the scheduled time
		if c.Status == models.CampaignInProgress && c.SourceType != models.SourceTypeClient && c.SourceType != models.SourceTypePage {
			go as.worker.LaunchCampaign(c)
		}
		JSONResponse(w, c, http.StatusCreated)
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
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		pagedJSONResponse(w, http.StatusOK, pp, cs.Campaigns, cs.Total)
	}
}

// DashboardStats returns lightweight aggregated stats for dashboard charts,
// avoiding the N+1 query problem of loading all campaign details.
func (as *Server) DashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	resp, err := models.GetDashboardStats(ctx.Get(r, "user_id").(int64))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, resp, http.StatusOK)
}

// Campaign returns details about the requested campaign. If the campaign is not
// valid, APICampaign returns null.
func (as *Server) Campaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetCampaign(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, c, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteCampaign(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting campaign"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign deleted successfully!"}, http.StatusOK)
	}
}

// CampaignResults returns just the results for a given campaign to
// significantly reduce the information returned.
func (as *Server) CampaignResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	pp := parsePagination(r)
	cr, err := models.GetCampaignResults(id, ctx.Get(r, "user_id").(int64), pp)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if r.Method == "GET" {
		JSONResponse(w, cr, http.StatusOK)
		return
	}
}

// CampaignResultsExport downloads the campaign results as a CSV file.
func (as *Server) CampaignResultsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	cr, err := models.GetCampaignResults(id, ctx.Get(r, "user_id").(int64), models.PageParams{})
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	fixedKeys := []string{"id", "smtp_id", "status", "ip", "latitude", "longitude", "send_date", "reported", "modified_date", "smtp_from_address", "email", "full_name", "position"}
	if len(cr.Results) > 0 {
		// Mirror the original frontend export, which derived the CSV columns from
		// Object.keys() on the first result record (i.e. the JSON field order).
		if keys, err := jsonKeys(cr.Results[0]); err == nil && len(keys) > 0 {
			fixedKeys = keys
		}
	}
	rows := make([]util.CSVRow, 0, len(cr.Results))
	for i := range cr.Results {
		row := util.CSVRow{Fixed: make([]interface{}, len(fixedKeys))}
		if m, err := resultMap(&cr.Results[i]); err == nil {
			for j, k := range fixedKeys {
				row.Fixed[j] = m[k]
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
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	cr, err := models.GetCampaignResults(id, ctx.Get(r, "user_id").(int64), models.PageParams{})
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	fixedKeys := []string{"campaign_id", "email", "time", "message", "details"}
	rows := make([]util.CSVRow, 0, len(cr.Events))
	for i := range cr.Events {
		e := &cr.Events[i]
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
				JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
			} else {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			}
			log.Error(err)
			return
		}
		JSONResponse(w, cs, http.StatusOK)
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
			JSONResponse(w, models.Response{Success: false, Message: "Error completing campaign"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign completed successfully!"}, http.StatusOK)
	}
}

// CampaignLaunch launches a scheduled or queued campaign immediately.
func (as *Server) CampaignLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	c, err := models.GetCampaign(id, uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if c.SourceType != models.SourceTypeEmail {
		JSONResponse(w, models.Response{Success: false, Message: "Only email campaigns can be launched"}, http.StatusBadRequest)
		return
	}
	if c.Status != models.CampaignScheduled && c.Status != models.CampaignQueued {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign is not in a launchable state"}, http.StatusBadRequest)
		return
	}
	err = models.LaunchCampaign(id, uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	// Launch via worker
	go as.worker.LaunchCampaign(c)
	JSONResponse(w, models.Response{Success: true, Message: "Campaign launched successfully!"}, http.StatusOK)
}
