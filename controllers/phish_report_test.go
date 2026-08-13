package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/util"
)

func createClientCampaign(t *testing.T, ctx *testContext) models.Campaign {
	c := models.Campaign{
		Name:       "Client drill",
		SourceType: models.SourceTypeClient,
		URL:        ctx.phishServer.URL,
	}
	c.ReportConfig = &models.ReportConfig{
		Fields: []models.ReportField{
			{Key: "ip", Label: "IP地址", Type: models.FieldTypeIP, Required: true},
			{Key: "mac", Label: "MAC地址", Type: models.FieldTypeMAC, Required: true},
			{Key: "username", Label: "用户名", Type: models.FieldTypeUsername},
		},
		DedupKey: "mac",
	}
	err := models.PostCampaign(&c, 1)
	if err != nil {
		t.Fatalf("error creating client campaign: %v", err)
	}
	return c
}

func createPageCampaign(t *testing.T, ctx *testContext) models.Campaign {
	p := models.Page{Name: "Fixed Page", HTML: "<html><body>欢迎参与演练</body></html>", UserId: 1}
	if err := models.PostPage(&p); err != nil {
		t.Fatalf("error posting page: %v", err)
	}
	c := models.Campaign{
		Name:       "Page drill",
		SourceType: models.SourceTypePage,
		URL:        ctx.phishServer.URL + "/fixed",
		Page:       p,
	}
	if err := models.PostCampaign(&c, 1); err != nil {
		t.Fatalf("error creating page campaign: %v", err)
	}
	// Sanity-check the default report config was persisted.
	if len(c.ReportConfig.Fields) == 0 {
		t.Fatalf("expected default report config, got none")
	}
	return c
}

func postReport(t *testing.T, ctx *testContext, campaignID int64, source string, key string, data map[string]interface{}) *http.Response {
	payload, _ := json.Marshal(map[string]interface{}{
		"campaign_id": campaignID,
		"source":      source,
		"data":        data,
		"_key":        key,
	})
	req, err := http.NewRequest("POST", ctx.phishServer.URL+"/api/report", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("error building report request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Simulate the cs client sending the key via header.
	req.Header.Set("X-Report-Key", key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error posting report: %v", err)
	}
	return resp
}

func TestClientReportFlow(t *testing.T) {
	ctx := setupTest(t)
	defer tearDown(t, ctx)
	c := createClientCampaign(t, ctx)

	// Generate a client code and ensure it embeds the report URL and key.
	rc, err := models.GetCampaignReportConfig(&c)
	if err != nil {
		t.Fatalf("error getting report config: %v", err)
	}
	key := models.ReportKey(c.Id, c.ReportSalt)
	fromAPI, err := util.GenerateClientCode(strings.TrimRight(c.URL, "/"), key, rc.DedupKey, c.Id, rc.Fields)
	if err != nil {
		t.Fatalf("error generating client code: %v", err)
	}
	for _, needle := range []string{"reportURL", ctx.phishServer.URL + "/api/report", key, "collectField"} {
		if !strings.Contains(fromAPI, needle) {
			t.Fatalf("generated client code missing %q", needle)
		}
	}

	// Valid report via the endpoint.
	resp := postReport(t, ctx, c.Id, models.SourceTypeClient, key, map[string]interface{}{
		"ip": "1.2.3.4", "mac": "00:11:22:33:44:55", "username": "zhangsan",
	})
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	reports, total, err := models.GetCampaignReports(c.Id, models.PageParams{})
	if err != nil || total != 1 || len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d (err=%v)", total, err)
	}
	if reports[0].Data["username"] != "zhangsan" {
		t.Fatalf("unexpected report data: %v", reports[0].Data)
	}

	// Idempotent - same dedup value is skipped.
	resp = postReport(t, ctx, c.Id, models.SourceTypeClient, key, map[string]interface{}{
		"ip": "1.2.3.5", "mac": "00:11:22:33:44:55", "username": "lisi",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for duplicate, got %d", resp.StatusCode)
	}
	_, total, err = models.GetCampaignReports(c.Id, models.PageParams{})
	if err != nil || total != 1 {
		t.Fatalf("expected still 1 report after duplicate, got %d (err=%v)", total, err)
	}

	// Invalid key is rejected.
	resp = postReport(t, ctx, c.Id, models.SourceTypeClient, "bogus-key", map[string]interface{}{
		"ip": "1.2.3.6", "mac": "00:11:22:33:44:66", "username": "wangwu",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid key, got %d", resp.StatusCode)
	}
}

func TestPageReportFlow(t *testing.T) {
	ctx := setupTest(t)
	defer tearDown(t, ctx)
	c := createPageCampaign(t, ctx)

	// The fixed URL serves the page HTML directly (no auto-injected form).
	resp, err := http.Get(ctx.phishServer.URL + "/fixed")
	if err != nil {
		t.Fatalf("error requesting fixed page: %v", err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for fixed page, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "欢迎参与演练") {
		t.Fatalf("fixed page missing expected HTML content")
	}

	key := models.ReportKey(c.Id, c.ReportSalt)
	// Valid page report using the key from the body.
	resp = postReport(t, ctx, c.Id, models.SourceTypePage, key, map[string]interface{}{
		"username": "zhangsan",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for page report, got %d", resp.StatusCode)
	}
	_, total, err := models.GetCampaignReports(c.Id, models.PageParams{})
	if err != nil || total != 1 {
		t.Fatalf("expected 1 page report, got %d (err=%v)", total, err)
	}

	// Invalid key rejected.
	resp = postReport(t, ctx, c.Id, models.SourceTypePage, "wrong", map[string]interface{}{
		"username": "lisi",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid page key, got %d", resp.StatusCode)
	}

	// Unknown campaign id rejected.
	resp = postReport(t, ctx, 99999, models.SourceTypePage, key, map[string]interface{}{"username": "a"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown campaign, got %d", resp.StatusCode)
	}
}

func adminAPIRequest(t *testing.T, ctx *testContext, path string) (*http.Response, []byte) {
	resp, err := http.Get(ctx.adminServer.URL + "/api" + path)
	if err != nil {
		t.Fatalf("error requesting admin API %s: %v", path, err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, body
}

func TestAdminReportEndpoints(t *testing.T) {
	ctx := setupTest(t)
	defer tearDown(t, ctx)
	c := createClientCampaign(t, ctx)
	pc := createPageCampaign(t, ctx)

	send := func(path string) (*http.Response, []byte) {
		req, err := http.NewRequest("GET", ctx.adminServer.URL+"/api"+path, nil)
		if err != nil {
			t.Fatalf("error building request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+ctx.apiKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("error requesting %s: %v", path, err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, body
	}

	// client code endpoint
	resp, body := send(fmt.Sprintf("/campaigns/%d/client/code", c.Id))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("client/code expected 200, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "collectField") {
		t.Fatalf("client/code response missing generated source: %s", body)
	}

	// page url endpoint
	resp, body = send(fmt.Sprintf("/campaigns/%d/page/url", pc.Id))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page/url expected 200, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "/fixed") {
		t.Fatalf("page/url response missing fixed url: %s", body)
	}

	// reports endpoint after a submission
	key := models.ReportKey(c.Id, c.ReportSalt)
	pre := postReport(t, ctx, c.Id, models.SourceTypeClient, key, map[string]interface{}{
		"ip": "10.0.0.8", "mac": "0a:0b:0c:0d:0e:0f",
	})
	pre.Body.Close()
	resp, body = send(fmt.Sprintf("/campaigns/%d/reports", c.Id))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reports expected 200, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "10.0.0.8") {
		t.Fatalf("reports response missing submitted data: %s", body)
	}
}
