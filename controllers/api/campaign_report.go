package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	// The Go client sends a fixed Go-http-client UA, which carries no signal,
	// so the user agent is only exported for fixed-page (browser) reports.
	// Client-reported data is passed through the shared writer, which renames
	// any key that collides with a fixed column (e.g. a reported "ip" against
	// the connection IP column becomes "ip1") so nothing is dropped.
	fixedKeys := []string{"id", "ip"}
	if c.SourceType == models.SourceTypePage {
		fixedKeys = append(fixedKeys, "user_agent")
	}
	fixedKeys = append(fixedKeys, "created_at")
	rows := make([]util.CSVRow, 0, len(reports))
	for _, re := range reports {
		fixed := []interface{}{re.Id, re.IP}
		if c.SourceType == models.SourceTypePage {
			fixed = append(fixed, re.UserAgent)
		}
		fixed = append(fixed, re.CreatedAt)
		rows = append(rows, util.CSVRow{Fixed: fixed, Data: re.Data})
	}
	writeCSVFile(w, c.Name, "reports", fixedKeys, rows)
}

// writeCSVFile sets the download headers for a CSV export and streams the rows
// through the shared CSV writer.
func writeCSVFile(w http.ResponseWriter, baseName, suffix string, fixedKeys []string, rows []util.CSVRow) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	safeName := strings.NewReplacer("\"", "_", "\\", "_", "\n", "_", "\r", "_").Replace(baseName)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.csv"`, safeName, suffix))
	if err := util.WriteCSV(w, fixedKeys, rows); err != nil {
		log.Error(err)
	}
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
