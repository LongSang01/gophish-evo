package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/util"
	"github.com/gorilla/mux"
)

// ClientCode returns the standalone Go source code for a client-type
// campaign. The operator downloads/copies this source and compiles it
// themselves with `go build`.
func (as *Server) ClientCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetCampaign(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if c.SourceType != models.SourceTypeClient {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign is not a client activity"}, http.StatusBadRequest)
		return
	}
	rc, err := models.GetCampaignReportConfig(&c)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	key := models.ReportKey(c.Id, c.ReportSalt)
	code, err := util.GenerateClientCode(strings.TrimRight(c.URL, "/"), key, rc.DedupKey, c.Id, rc.Fields)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to generate client code"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "ok", Data: code}, http.StatusOK)
}

// PageURL returns the fixed URL for a page-type campaign. The server does not
// generate a QR code; the operator creates one themselves.
func (as *Server) PageURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetCampaign(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if c.SourceType != models.SourceTypePage {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign is not a fixed page activity"}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "ok", Data: c.URL}, http.StatusOK)
}

// CampaignReports returns the report records collected for a campaign.
func (as *Server) CampaignReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	// Verify the campaign exists and belongs to the user.
	if _, err := models.GetCampaign(id, ctx.Get(r, "user_id").(int64)); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	pp := parsePagination(r)
	reports, total, err := models.GetCampaignReports(id, pp)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	pagedJSONResponse(w, http.StatusOK, pp, reports, total)
}

// CampaignReportsExport downloads the collected report records as a CSV file.
// Column set is the union of all dynamic field keys across the records.
func (as *Server) CampaignReportsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetCampaign(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	reports, _, err := models.GetCampaignReports(id, models.PageParams{})
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	// Build the union of dynamic keys. The Go client sends a fixed
	// Go-http-client UA, which carries no signal, so it is only exported for
	// fixed-page (browser) reports.
	keys := []string{"id", "source", "ip"}
	if c.SourceType == models.SourceTypePage {
		keys = append(keys, "user_agent")
	}
	keys = append(keys, "created_at")
	seen := map[string]bool{}
	for _, k := range keys {
		seen[k] = true
	}
	for _, re := range reports {
		for k := range re.Data {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	safeName := strings.NewReplacer("\"", "_", "\\", "_", "\n", "_", "\r", "_").Replace(c.Name)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-reports.csv"`, safeName))
	w.Write([]byte("\uFEFF")) // UTF-8 BOM for Excel compatibility
	cw := csv.NewWriter(w)
	_ = cw.Write(keys)
	colIndex := make(map[string]int, len(keys))
	for i, k := range keys {
		colIndex[k] = i
	}
	baseIdx := colIndex["created_at"] + 1
	for _, re := range reports {
		row := make([]string, len(keys))
		row[colIndex["id"]] = strconv.FormatInt(re.Id, 10)
		row[colIndex["source"]] = re.Source
		row[colIndex["ip"]] = re.IP
		if i, ok := colIndex["user_agent"]; ok {
			row[i] = re.UserAgent
		}
		row[colIndex["created_at"]] = re.CreatedAt.Format(time.RFC3339)
		for i, k := range keys[baseIdx:] {
			v := ""
			if val, ok := re.Data[k]; ok {
				switch t := val.(type) {
				case string:
					v = t
				case fmt.Stringer:
					v = t.String()
				default:
					v = fmt.Sprintf("%v", t)
				}
			}
			row[i+baseIdx] = sanitizeCSVValue(v)
		}
		_ = cw.Write(row)
	}
	cw.Flush()
}

// sanitizeCSVValue prevents CSV formula injection by prefixing values that
// start with characters interpreted as formula indicators by spreadsheet
// applications (=, +, -, @, tab, carriage return).
func sanitizeCSVValue(v string) string {
	if len(v) > 0 {
		switch v[0] {
		case '=', '+', '-', '@', '\t', '\r':
			return "'" + v
		}
	}
	return v
}

// CampaignReportSummary returns an aggregated view of page campaign reports,
// merging submitted reports with click statistics. For page-type campaigns,
// each unique visitor (identified by vid cookie) appears as a single row
// showing both their submission and click activity.
func (as *Server) CampaignReportSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetCampaign(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if c.SourceType != models.SourceTypePage {
		JSONResponse(w, models.Response{Success: false, Message: "Report summary is only available for page-type campaigns"}, http.StatusBadRequest)
		return
	}
	pp := parsePagination(r)
	reports, total, err := models.GetCampaignReportSummary(id, pp)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	pagedJSONResponse(w, http.StatusOK, pp, reports, total)
}
