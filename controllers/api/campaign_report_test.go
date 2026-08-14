package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophish/gophish/models"
)

func TestCampaignReportsExportClientKeepsReportedIP(t *testing.T) {
	tc := setupTest(t)

	c := models.Campaign{
		Name:       "Client export drill",
		SourceType: models.SourceTypeClient,
		URL:        "http://127.0.0.1:3333",
		ReportConfig: &models.ReportConfig{
			Fields: []models.ReportField{
				{Key: "ip", Label: "IP地址", Type: models.FieldTypeIP},
				{Key: "mac", Label: "MAC地址", Type: models.FieldTypeMAC},
			},
			DedupKey: "mac",
		},
	}
	if err := models.PostCampaign(&c, 1); err != nil {
		t.Fatalf("error creating client campaign: %v", err)
	}
	rc, err := models.GetCampaignReportConfig(&c)
	if err != nil {
		t.Fatalf("error getting report config: %v", err)
	}
	// The client reports its machine IP under the "ip" key; the connection IP
	// is captured server-side separately.
	if _, err := models.SaveReportExtBatch(c.Id, []models.Map{
		{"ip": "eth0:192.168.1.100", "mac": "00:11:22:33:44:55"},
	}, rc, "203.0.113.9", "Go-http-client/1.1", ""); err != nil {
		t.Fatalf("error saving report: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/reports/export", c.Id), nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"ip1",                // reported machine IP column, de-duplicated against the connection IP column
		"eth0:192.168.1.100", // reported machine IP value
		"00:11:22:33:44:55",  // mac still exported
		"203.0.113.9",        // connection IP kept in the ip column
	} {
		if !strings.Contains(body, want) {
			t.Errorf("export missing %q, got:\n%s", want, body)
		}
	}
}

func TestCampaignReportsExportDedupCustomFields(t *testing.T) {
	tc := setupTest(t)

	c := models.Campaign{
		Name:       "Client export dedup drill",
		SourceType: models.SourceTypeClient,
		URL:        "http://127.0.0.1:3333",
		ReportConfig: &models.ReportConfig{
			Fields: []models.ReportField{
				{Key: "ip", Label: "IP地址", Type: models.FieldTypeIP},
				{Key: "mac", Label: "MAC地址", Type: models.FieldTypeMAC},
				{Key: "created_at", Label: "自定义时间", Type: models.FieldTypeCustom},
			},
			DedupKey: "mac",
		},
	}
	if err := models.PostCampaign(&c, 1); err != nil {
		t.Fatalf("error creating client campaign: %v", err)
	}
	rc, err := models.GetCampaignReportConfig(&c)
	if err != nil {
		t.Fatalf("error getting report config: %v", err)
	}
	if _, err := models.SaveReportExtBatch(c.Id, []models.Map{
		{"ip": "eth0:10.0.0.5", "mac": "de:ad:be:ef:00:01", "created_at": "2026-08-14"},
	}, rc, "198.51.100.7", "Go-http-client/1.1", ""); err != nil {
		t.Fatalf("error saving report: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/reports/export", c.Id), nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"ip1",           // reported ip de-duplicated
		"eth0:10.0.0.5", // reported ip value
		"created_at1",   // custom field colliding with the static created_at column
		"2026-08-14",    // custom field value
	} {
		if !strings.Contains(body, want) {
			t.Errorf("export missing %q, got:\n%s", want, body)
		}
	}
}

func TestCampaignReportsExportPageUsesConnectionIP(t *testing.T) {
	tc := setupTest(t)

	p := models.Page{Name: "Fixed Export Page", HTML: "<html></html>", UserId: 1}
	if err := models.PostPage(&p); err != nil {
		t.Fatalf("error creating page: %v", err)
	}
	c := models.Campaign{
		Name:       "Page export drill",
		SourceType: models.SourceTypePage,
		URL:        "http://127.0.0.1:3333/fixed",
		Page:       p,
	}
	if err := models.PostCampaign(&c, 1); err != nil {
		t.Fatalf("error creating page campaign: %v", err)
	}
	rc, err := models.GetCampaignReportConfig(&c)
	if err != nil {
		t.Fatalf("error getting report config: %v", err)
	}
	if _, err := models.SaveReportExtBatch(c.Id, []models.Map{
		{"username": "zhangsan"},
	}, rc, "203.0.113.7", "Mozilla/5.0", ""); err != nil {
		t.Fatalf("error saving report: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/reports/export", c.Id), nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"203.0.113.7", // connection IP in the ip column
		"zhangsan",
		"user_agent",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("export missing %q, got:\n%s", want, body)
		}
	}
}

func createEmailCampaignForExport(t *testing.T, tc *testContext, name string) models.Campaign {
	createTestData(t)
	c := models.Campaign{Name: name}
	c.Template = models.Template{Name: "Test Template"}
	c.Page = models.Page{Name: "Test Page"}
	c.SMTP = models.SMTP{Name: "Test SMTP"}
	c.Groups = []models.Group{{Name: "Test Group"}}
	if err := models.PostCampaign(&c, 1); err != nil {
		t.Fatalf("error creating campaign: %v", err)
	}
	return c
}

func TestCampaignResultsExport(t *testing.T) {
	tc := setupTest(t)
	c := createEmailCampaignForExport(t, tc, "Results Export Test")

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/results/export", c.Id), nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"id,smtp_id,status,ip,latitude,longitude,send_date,reported,modified_date,smtp_from_address,email,full_name,position",
		"test1@example.com",
		"test2@example.com",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("export missing %q, got:\n%s", want, body)
		}
	}
}

func TestCampaignEventsExport(t *testing.T) {
	tc := setupTest(t)
	c := createEmailCampaignForExport(t, tc, "Events Export Test")

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/events/export", c.Id), nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"campaign_id,email,time,message,details",
		"Campaign Created",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("export missing %q, got:\n%s", want, body)
		}
	}
}
