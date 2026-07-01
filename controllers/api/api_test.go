package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophish/gophish/auth"
	"github.com/gophish/gophish/config"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

type testContext struct {
	apiKey    string
	config    *config.Config
	apiServer *Server
	admin     models.User
}

func setupTest(t *testing.T) *testContext {
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../../db/db_sqlite3/migrations/",
	}
	err := models.Setup(conf)
	if err != nil {
		t.Fatalf("Failed creating database: %v", err)
	}
	tc := &testContext{}
	tc.config = conf
	u, err := models.GetUser(1)
	if err != nil {
		t.Fatalf("error getting admin user: %v", err)
	}
	tc.apiKey = u.ApiKey
	tc.admin = u
	tc.apiServer = NewServer()
	return tc
}

func setKnownPassword(t *testing.T, tc *testContext) {
	hash, err := auth.GeneratePasswordHash("gophish")
	if err != nil {
		t.Fatalf("error generating password hash: %v", err)
	}
	tc.admin.Hash = hash
	tc.admin.PasswordChangeRequired = false
	err = models.PutUser(&tc.admin)
	if err != nil {
		t.Fatalf("error setting admin password: %v", err)
	}
}

func createTestData(t *testing.T) {
	group := models.Group{Name: "Test Group"}
	group.Targets = []models.Target{
		{BaseRecipient: models.BaseRecipient{Email: "test1@example.com", FullName: "First Example"}},
		{BaseRecipient: models.BaseRecipient{Email: "test2@example.com", FullName: "Second Example"}},
	}
	group.UserId = 1
	models.PostGroup(&group)

	template := models.Template{Name: "Test Template"}
	template.Subject = "Test subject"
	template.Text = "Text text"
	template.HTML = "<html>Test</html>"
	template.UserId = 1
	models.PostTemplate(&template)

	p := models.Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	models.PostPage(&p)

	smtp := models.SMTP{Name: "Test SMTP"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	models.PostSMTP(&smtp)
}

// ==================== Auth Tests ====================

func TestLoginSuccess(t *testing.T) {
	tc := setupTest(t)
	setKnownPassword(t, tc)

	body := `{"username":"admin","password":"gophish"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.apiServer.Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := LoginResponse{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.User == nil {
		t.Fatal("expected user in response")
	}
	if resp.User.Username != "admin" {
		t.Fatalf("expected username 'admin', got %s", resp.User.Username)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	tc := setupTest(t)
	setKnownPassword(t, tc)

	body := `{"username":"admin","password":"wrongpassword"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.apiServer.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestLoginWrongMethod(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()
	tc.apiServer.Login(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", w.Code)
	}
}

func TestLogout(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	tc.apiServer.Logout(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestLogoutWrongMethod(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	tc.apiServer.Logout(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", w.Code)
	}
}

func TestGetCurrentUser(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.GetCurrentUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	u := models.User{}
	json.NewDecoder(w.Body).Decode(&u)
	if u.Id != tc.admin.Id {
		t.Fatalf("expected user id %d got %d", tc.admin.Id, u.Id)
	}
}

func TestChangePassword(t *testing.T) {
	tc := setupTest(t)
	setKnownPassword(t, tc)

	body := `{"current_password":"gophish","new_password":"newpassword123","confirm_password":"newpassword123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.ChangePassword(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	tc := setupTest(t)
	setKnownPassword(t, tc)

	body := `{"current_password":"wrongpassword","new_password":"newpassword123","confirm_password":"newpassword123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.ChangePassword(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestResetPasswordRequired(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/reset-password-required", nil)
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.ResetPasswordRequired(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	m := map[string]bool{}
	json.NewDecoder(w.Body).Decode(&m)
	if _, ok := m["password_change_required"]; !ok {
		t.Fatal("expected password_change_required field in response")
	}
}

// ==================== Campaign Tests ====================

func TestGetCampaigns(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	c := models.Campaign{Name: "Test Campaign"}
	c.UserId = 1
	c.Template = models.Template{Name: "Test Template"}
	c.Page = models.Page{Name: "Test Page"}
	c.SMTP = models.SMTP{Name: "Test SMTP"}
	c.Groups = []models.Group{{Name: "Test Group"}}
	models.PostCampaign(&c, 1)

	r := httptest.NewRequest(http.MethodGet, "/api/campaigns/", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Campaigns(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	cs := []models.Campaign{}
	json.NewDecoder(w.Body).Decode(&cs)
	if len(cs) == 0 {
		t.Fatal("expected at least one campaign")
	}
}

func TestCreateCampaign(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"name":"New Campaign","template":{"name":"Test Template"},"page":{"name":"Test Page"},"smtp":{"name":"Test SMTP"},"groups":[{"name":"Test Group"}]}`
	r := httptest.NewRequest(http.MethodPost, "/api/campaigns/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Campaigns(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	c := models.Campaign{}
	json.NewDecoder(w.Body).Decode(&c)
	if c.Name != "New Campaign" {
		t.Fatalf("expected name 'New Campaign' got %s", c.Name)
	}
	if c.Id == 0 {
		t.Fatal("expected non-zero campaign id")
	}
}

func TestCreateCampaignInvalidJSON(t *testing.T) {
	tc := setupTest(t)

	body := `{invalid json}`
	r := httptest.NewRequest(http.MethodPost, "/api/campaigns/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Campaigns(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCampaignComplete(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	c := models.Campaign{Name: "Complete Test"}
	c.UserId = 1
	c.Template = models.Template{Name: "Test Template"}
	c.Page = models.Page{Name: "Test Page"}
	c.SMTP = models.SMTP{Name: "Test SMTP"}
	c.Groups = []models.Group{{Name: "Test Group"}}
	models.PostCampaign(&c, 1)

	url := fmt.Sprintf("/api/campaigns/%d/complete", c.Id)
	r := httptest.NewRequest(http.MethodGet, url, nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestCampaignLaunch(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	c := models.Campaign{Name: "Launch Test"}
	c.UserId = 1
	c.Template = models.Template{Name: "Test Template"}
	c.Page = models.Page{Name: "Test Page"}
	c.SMTP = models.SMTP{Name: "Test SMTP"}
	c.Groups = []models.Group{{Name: "Test Group"}}
	models.PostCampaign(&c, 1)
	c.UpdateStatus(models.CampaignScheduled)

	url := fmt.Sprintf("/api/campaigns/%d/launch", c.Id)
	r := httptest.NewRequest(http.MethodPost, url, nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

// ==================== Group Tests ====================

func TestGetGroups(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Groups(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	gs := []models.Group{}
	json.NewDecoder(w.Body).Decode(&gs)
	if len(gs) == 0 {
		t.Fatal("expected at least one group")
	}
}

func TestCreateGroup(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"New Group","targets":[{"email":"new@example.com","full_name":"New User","position":"Tester"}]}`
	r := httptest.NewRequest(http.MethodPost, "/api/groups/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Groups(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	g := models.Group{}
	json.NewDecoder(w.Body).Decode(&g)
	if g.Name != "New Group" {
		t.Fatalf("expected name 'New Group' got %s", g.Name)
	}
	if g.Id == 0 {
		t.Fatal("expected non-zero group id")
	}
}

func TestCreateGroupDuplicateName(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"name":"Test Group","targets":[{"email":"dup@example.com"}]}`
	r := httptest.NewRequest(http.MethodPost, "/api/groups/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Groups(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestDeleteGroup(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	url := "/api/groups/1"
	r := httptest.NewRequest(http.MethodDelete, url, nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestModifyGroup(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"id":1,"name":"Modified Group","targets":[{"email":"modified@example.com","full_name":"Modified"}]}`
	url := "/api/groups/1"
	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	g := models.Group{}
	json.NewDecoder(w.Body).Decode(&g)
	if g.Name != "Modified Group" {
		t.Fatalf("expected name 'Modified Group' got %s", g.Name)
	}
}

// ==================== Template Tests ====================

func TestGetTemplates(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	r := httptest.NewRequest(http.MethodGet, "/api/templates/", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Templates(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	ts := []models.Template{}
	json.NewDecoder(w.Body).Decode(&ts)
	if len(ts) == 0 {
		t.Fatal("expected at least one template")
	}
}

func TestCreateTemplate(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"New Template","subject":"Test Subject","text":"Hello {{.FirstName}}","html":"<html>Hello {{.FirstName}}</html>"}`
	r := httptest.NewRequest(http.MethodPost, "/api/templates/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Templates(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	tmpl := models.Template{}
	json.NewDecoder(w.Body).Decode(&tmpl)
	if tmpl.Name != "New Template" {
		t.Fatalf("expected name 'New Template' got %s", tmpl.Name)
	}
}

func TestCreateTemplateDuplicateName(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"name":"Test Template","subject":"Sub","text":"Body"}`
	r := httptest.NewRequest(http.MethodPost, "/api/templates/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Templates(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestDeleteTemplate(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	url := "/api/templates/1"
	r := httptest.NewRequest(http.MethodDelete, url, nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestModifyTemplate(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"id":1,"name":"Modified Template","subject":"Modified","text":"Modified body","html":"<html>Modified</html>"}`
	url := "/api/templates/1"
	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	tmpl := models.Template{}
	json.NewDecoder(w.Body).Decode(&tmpl)
	if tmpl.Name != "Modified Template" {
		t.Fatalf("expected name 'Modified Template' got %s", tmpl.Name)
	}
}

// ==================== Page Tests ====================

func TestGetPages(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	r := httptest.NewRequest(http.MethodGet, "/api/pages/", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Pages(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	ps := []models.Page{}
	json.NewDecoder(w.Body).Decode(&ps)
	if len(ps) == 0 {
		t.Fatal("expected at least one page")
	}
}

func TestCreatePage(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"New Page","html":"<html>New Landing Page</html>","capture_credentials":true}`
	r := httptest.NewRequest(http.MethodPost, "/api/pages/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Pages(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	p := models.Page{}
	json.NewDecoder(w.Body).Decode(&p)
	if p.Name != "New Page" {
		t.Fatalf("expected name 'New Page' got %s", p.Name)
	}
}

func TestCreatePageDuplicateName(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"name":"Test Page","html":"<html>Duplicate</html>"}`
	r := httptest.NewRequest(http.MethodPost, "/api/pages/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.Pages(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
}

func TestDeletePage(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	url := "/api/pages/1"
	r := httptest.NewRequest(http.MethodDelete, url, nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestModifyPage(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"id":1,"name":"Modified Page","html":"<html>Modified</html>"}`
	url := "/api/pages/1"
	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	p := models.Page{}
	json.NewDecoder(w.Body).Decode(&p)
	if p.Name != "Modified Page" {
		t.Fatalf("expected name 'Modified Page' got %s", p.Name)
	}
}

// ==================== SMTP Tests ====================

func TestGetSendingProfiles(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	r := httptest.NewRequest(http.MethodGet, "/api/smtp/", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.SendingProfiles(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	ss := []models.SMTP{}
	json.NewDecoder(w.Body).Decode(&ss)
	if len(ss) == 0 {
		t.Fatal("expected at least one SMTP profile")
	}
}

func TestCreateSendingProfile(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"New SMTP","host":"mail.example.com:587","from_address":"phish@example.com","username":"user","password":"pass","ignore_cert_errors":true}`
	r := httptest.NewRequest(http.MethodPost, "/api/smtp/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.SendingProfiles(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	s := models.SMTP{}
	json.NewDecoder(w.Body).Decode(&s)
	if s.Name != "New SMTP" {
		t.Fatalf("expected name 'New SMTP' got %s", s.Name)
	}
}

func TestCreateSendingProfileMissingHost(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"Bad SMTP","from_address":"test@test.com"}`
	r := httptest.NewRequest(http.MethodPost, "/api/smtp/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.SendingProfiles(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSendingProfile(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	url := "/api/smtp/1"
	r := httptest.NewRequest(http.MethodDelete, url, nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestModifySendingProfile(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	body := `{"id":1,"name":"Modified SMTP","host":"modified.com:25","from_address":"mod@example.com"}`
	url := "/api/smtp/1"
	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	s := models.SMTP{}
	json.NewDecoder(w.Body).Decode(&s)
	if s.Name != "Modified SMTP" {
		t.Fatalf("expected name 'Modified SMTP' got %s", s.Name)
	}
}

// ==================== API Key Reset Test ====================

func TestResetAPIKey(t *testing.T) {
	tc := setupTest(t)
	oldKey := tc.admin.ApiKey

	r := httptest.NewRequest(http.MethodPost, "/api/reset", nil)
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Reset(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
	if resp.Data == oldKey {
		t.Fatal("expected new API key to differ from old one")
	}
}

// ==================== Webhook Tests ====================

func TestGetWebhooks(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/webhooks/", nil)
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Webhooks(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestCreateWebhook(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"Test Webhook","url":"http://example.com/hook","secret":"mysecret","is_active":true}`
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Webhooks(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	wh := models.Webhook{}
	json.NewDecoder(w.Body).Decode(&wh)
	if wh.Name != "Test Webhook" {
		t.Fatalf("expected name 'Test Webhook' got %s", wh.Name)
	}
}

func TestCreateWebhookInvalid(t *testing.T) {
	tc := setupTest(t)

	body := `{"name":"","url":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Webhooks(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestDeleteWebhook(t *testing.T) {
	tc := setupTest(t)

	wh := models.Webhook{Name: "To Delete", URL: "http://example.com/del"}
	models.PostWebhook(&wh)

	url := fmt.Sprintf("/api/webhooks/%d", wh.Id)
	r := httptest.NewRequest(http.MethodDelete, url, nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	resp := models.Response{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success=true, got message=%s", resp.Message)
	}
}

func TestModifyWebhook(t *testing.T) {
	tc := setupTest(t)

	wh := models.Webhook{Name: "To Modify", URL: "http://example.com/mod", Secret: "oldsecret"}
	models.PostWebhook(&wh)

	body := `{"name":"Modified Webhook","url":"http://modified.com/hook","secret":"newsecret","is_active":false}`
	url := fmt.Sprintf("/api/webhooks/%d", wh.Id)
	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	modified := models.Webhook{}
	json.NewDecoder(w.Body).Decode(&modified)
	if modified.Name != "Modified Webhook" {
		t.Fatalf("expected name 'Modified Webhook' got %s", modified.Name)
	}
}

// ==================== IMAP Tests ====================

func TestGetIMAP(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/imap/", nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

// ==================== Summary Tests ====================

func TestCampaignsSummary(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	c := models.Campaign{Name: "Summary Test"}
	c.UserId = 1
	c.Template = models.Template{Name: "Test Template"}
	c.Page = models.Page{Name: "Test Page"}
	c.SMTP = models.SMTP{Name: "Test SMTP"}
	c.Groups = []models.Group{{Name: "Test Group"}}
	models.PostCampaign(&c, 1)

	r := httptest.NewRequest(http.MethodGet, "/api/campaigns/summary", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.CampaignsSummary(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	cs := models.CampaignSummaries{}
	json.NewDecoder(w.Body).Decode(&cs)
	if cs.Total < 1 {
		t.Fatal("expected at least one campaign in summary")
	}
}

func TestGroupsSummary(t *testing.T) {
	tc := setupTest(t)
	createTestData(t)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/summary", nil)
	r = ctx.Set(r, "user_id", int64(1))
	w := httptest.NewRecorder()
	tc.apiServer.GroupsSummary(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	gs := models.GroupSummaries{}
	json.NewDecoder(w.Body).Decode(&gs)
	if gs.Total < 1 {
		t.Fatal("expected at least one group in summary")
	}
}

// ==================== Import Tests ====================

func TestSiteImportBaseHref(t *testing.T) {
	tc := setupTest(t)
	h := "<html><head></head><body><img src=\"/test.png\"/></body></html>"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(h))
	}))
	defer ts.Close()

	body := fmt.Sprintf(`{"url":"%s","include_resources":false}`, ts.URL)
	r := httptest.NewRequest(http.MethodPost, "/api/import/site", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.apiServer.ImportSite(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	cs := cloneResponse{}
	json.NewDecoder(w.Body).Decode(&cs)
	if cs.HTML == "" {
		t.Fatal("expected non-empty HTML in response")
	}
}

func TestImportEmail(t *testing.T) {
	tc := setupTest(t)

	rawEmail := "From: sender@example.com\r\nTo: target@example.com\r\nSubject: Test Email\r\n\r\nHello, this is a test email."
	body := fmt.Sprintf(`{"content":%q,"convert_links":false}`, rawEmail)
	r := httptest.NewRequest(http.MethodPost, "/api/import/email", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.apiServer.ImportEmail(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	er := emailResponse{}
	json.NewDecoder(w.Body).Decode(&er)
	if er.Subject != "Test Email" {
		t.Fatalf("expected subject 'Test Email' got %s", er.Subject)
	}
}

func TestImportEmailConvertLinks(t *testing.T) {
	tc := setupTest(t)

	rawEmail := "From: sender@example.com\r\nTo: target@example.com\r\nSubject: Phish\r\nContent-Type: text/html\r\n\r\n<html><body><a href=\"http://evil.com\">Click</a></body></html>"
	body := fmt.Sprintf(`{"content":%q,"convert_links":true}`, rawEmail)
	r := httptest.NewRequest(http.MethodPost, "/api/import/email", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.apiServer.ImportEmail(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	er := emailResponse{}
	json.NewDecoder(w.Body).Decode(&er)
	if er.HTML == "" {
		t.Fatal("expected non-empty HTML in response")
	}
}

// ==================== NotFound Tests ====================

func TestGetCampaignNotFound(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/campaigns/999", nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetGroupNotFound(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/999", nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTemplateNotFound(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/templates/999", nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPageNotFound(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/pages/999", nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSMTPNotFound(t *testing.T) {
	tc := setupTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/smtp/999", nil)
	r.Header.Set("Authorization", "Bearer "+tc.apiKey)
	w := httptest.NewRecorder()
	tc.apiServer.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}
