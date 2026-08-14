package controllers

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/gophish/gophish/controllers/api"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// ReportRequest is the payload accepted by the /api/report endpoint. Both the
// client module and the fixed-page module report through this endpoint.
type ReportRequest struct {
	CampaignID int64        `json:"campaign_id"`
	Source     string       `json:"source"`
	Data       models.Map   `json:"data"`
	DataList   []models.Map `json:"data_list"`
	Key        string       `json:"_key"`
}

// parseReportRequest reads and validates the JSON body of a report request,
// supporting both a single map and a batch list. campaign_id may arrive as a
// JSON number or as a string (older client builds); both are accepted.
func parseReportRequest(r *http.Request) (*ReportRequest, error) {
	req := &ReportRequest{}
	var rawID json.RawMessage
	var body struct {
		CampaignID *json.RawMessage `json:"campaign_id"`
		Source     string           `json:"source"`
		Data       models.Map       `json:"data"`
		DataList   []models.Map     `json:"data_list"`
		Key        string           `json:"_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.CampaignID != nil {
		rawID = *body.CampaignID
	}
	req.Source = body.Source
	req.Data = body.Data
	req.DataList = body.DataList
	req.Key = body.Key

	// Parse campaign_id: try number first, then string.
	var numVal float64
	if err := json.Unmarshal(rawID, &numVal); err == nil {
		req.CampaignID = int64(numVal)
	} else {
		var strVal string
		if err := json.Unmarshal(rawID, &strVal); err != nil {
			return nil, ErrInvalidRequest
		}
		v, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return nil, err
		}
		req.CampaignID = v
	}

	if req.Source != models.SourceTypeClient && req.Source != models.SourceTypePage {
		return nil, ErrInvalidRequest
	}
	if req.CampaignID <= 0 {
		return nil, ErrInvalidRequest
	}
	if req.DataList == nil && req.Data == nil {
		return nil, ErrInvalidRequest
	}
	// The page module sends _key in the body; the client sends the key in the
	// X-Report-Key header. Prefer the header.
	if key := r.Header.Get("X-Report-Key"); key != "" {
		req.Key = key
	}
	return req, nil
}

func extractClientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ReportExtHandler is a public report endpoint used by both the fishing
// client and the fixed-page module. It authenticates the request with a
// per-campaign report key, then stores the reported data.
func (ps *PhishingServer) ReportExtHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		api.JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	req, err := parseReportRequest(r)
	if err != nil {
		api.JSONResponse(w, models.Response{Success: false, Message: "Invalid report request"}, http.StatusBadRequest)
		return
	}
	c, err := models.GetCampaignForReport(req.CampaignID)
	if err != nil {
		api.JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if c.SourceType != models.SourceTypeClient && c.SourceType != models.SourceTypePage {
		api.JSONResponse(w, models.Response{Success: false, Message: "Campaign does not support reporting"}, http.StatusBadRequest)
		return
	}
	if !models.ValidateReportKey(c.Id, c.ReportSalt, req.Key) {
		api.JSONResponse(w, models.Response{Success: false, Message: http.StatusText(http.StatusUnauthorized)}, http.StatusUnauthorized)
		return
	}
	rc, err := models.GetCampaignReportConfig(c)
	if err != nil {
		api.JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	ip := extractClientIP(r)
	ua := r.Header.Get("User-Agent")

	records := req.DataList
	if len(records) == 0 && req.Data != nil {
		records = []models.Map{req.Data}
	}
	if _, err := models.SaveReportExtBatch(c.Id, req.Source, records, rc, ip, ua, ""); err != nil {
		log.Error(err)
		api.JSONResponse(w, models.Response{Success: false, Message: "Failed to store report"}, http.StatusInternalServerError)
		return
	}
	api.JSONResponse(w, models.Response{Success: true, Message: "Report received"}, http.StatusOK)
}

// renderFixedPage serves the HTML of a page-type campaign at its fixed URL.
// For GET requests, a hidden capture form is injected if the page HTML does not
// already contain one, mirroring the email landing page flow. Form submissions
// (POST) are stored as "Submitted Data" events on the campaign timeline.
//
// Visitor tracking: on the first GET, a _vid cookie is set with a random
// identifier. Every GET increments an in-memory click counter keyed by
// (campaign_id, vid). The counter is periodically flushed to the
// page_click_stats table by a background goroutine. On POST, the vid from
// the cookie is stored alongside the submitted data in reports_ext.
func (ps *PhishingServer) renderFixedPage(w http.ResponseWriter, r *http.Request, c *models.Campaign, urlPath string) {
	// Reject requests for completed campaigns – mirrors the email flow where
	// setupContext returns ErrCampaignComplete for finished campaigns.
	if c.Status == models.CampaignComplete {
		http.NotFound(w, r)
		return
	}
	p, err := models.GetPage(c.PageId, c.UserId)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// --- Visitor ID cookie management (read or set) ---
	vid := readOrCreateVisitorID(w, r, urlPath)

	if r.Method == http.MethodGet {
		// Record a page open in the in-memory counter.
		ip := extractClientIP(r)
		models.ClickCounter.Incr(c.Id, vid, ip)
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			log.Error(err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		ip := extractClientIP(r)
		ua := r.Header.Get("User-Agent")

		// Persist form data to reports_ext (the single source of truth for
		// page-type campaigns, same as client-type campaigns).
		data := models.Map{}
		for k, v := range r.Form {
			if len(v) > 0 {
				data[k] = v[0]
			}
		}
		if len(data) > 0 {
			rc, err := models.GetCampaignReportConfig(c)
			if err != nil {
				log.Error(err)
			} else {
				if _, err := models.SaveReportExtBatch(c.Id, models.SourceTypePage, []models.Map{data}, rc, ip, ua, vid); err != nil {
					log.Error(err)
				}
			}
		}

		if p.RedirectURL != "" {
			http.Redirect(w, r, p.RedirectURL, http.StatusFound)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(p.HTML))
}

// visitorIDCookieName is the name of the cookie used to identify unique
// visitors to page-type campaigns.
const visitorIDCookieName = "_vid"

// readOrCreateVisitorID reads the visitor ID from the request cookie, or
// generates a new one and sets it on the response. The cookie is scoped to
// the campaign path with HttpOnly and SameSite=Lax for security.
func readOrCreateVisitorID(w http.ResponseWriter, r *http.Request, urlPath string) string {
	// Try to read existing cookie.
	if cookie, err := r.Cookie(visitorIDCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Generate a new visitor ID.
	vid, err := models.GenerateVisitorID()
	if err != nil {
		log.Errorf("Failed to generate visitor ID: %v", err)
		return ""
	}

	// Set cookie: HttpOnly, SameSite=Lax, scoped to the campaign path, 1 year expiry.
	cookie := &http.Cookie{
		Name:     visitorIDCookieName,
		Value:    vid,
		Path:     urlPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 60 * 60, // 1 year
	}
	http.SetCookie(w, cookie)

	return vid
}
