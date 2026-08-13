package util

import (
	"strings"
	"testing"

	"github.com/gophish/gophish/models"
)

func TestGenerateClientCodeBasic(t *testing.T) {
	fields := []models.ReportField{
		{Key: "ip", Label: "IP地址", Type: models.FieldTypeIP, Required: true},
		{Key: "mac", Label: "MAC地址", Type: models.FieldTypeMAC, Required: true},
		{Key: "username", Label: "用户名", Type: models.FieldTypeUsername},
		{Key: "hostname", Label: "主机名", Type: models.FieldTypeHostname},
	}
	code, err := GenerateClientCode("http://example.com", "test-key-123", "mac", 42, fields)
	if err != nil {
		t.Fatalf("GenerateClientCode returned error: %v", err)
	}

	// Verify essential components are present.
	checks := []string{
		"reportURL",
		"http://example.com/api/report",
		"test-key-123",
		"campaignID",
		"42",
		"collectField",
		"\"ip\"",
		"\"mac\"",
		"\"username\"",
		"\"hostname\"",
		"nonLoopbackIPs",
		"nonLoopbackMACs",
		"user.Current",
		"os.Hostname",
		"X-Report-Key",
		"30 * time.Second",
	}
	for _, needle := range checks {
		if !strings.Contains(code, needle) {
			t.Errorf("generated code missing %q\n---\n%s", needle, code)
		}
	}
}

func TestGenerateClientCodeCustomField(t *testing.T) {
	fields := []models.ReportField{
		{Key: "department", Label: "部门", Type: models.FieldTypeCustom},
	}
	code, err := GenerateClientCode("http://localhost:8080", "key", "department", 1, fields)
	if err != nil {
		t.Fatalf("GenerateClientCode returned error: %v", err)
	}
	if !strings.Contains(code, "\"department\"") {
		t.Errorf("generated code missing custom field key")
	}
	if !strings.Contains(code, "TODO") {
		t.Errorf("generated code should contain TODO for custom field")
	}
}

func TestGenerateClientCodeNoFields(t *testing.T) {
	code, err := GenerateClientCode("http://example.com", "key", "", 1, nil)
	if err != nil {
		t.Fatalf("GenerateClientCode returned error: %v", err)
	}
	if !strings.Contains(code, "reportURL") {
		t.Errorf("generated code should still be valid with no fields")
	}
}

func TestGenerateClientCodeEmptyBaseURL(t *testing.T) {
	code, err := GenerateClientCode("", "key", "mac", 1, []models.ReportField{
		{Key: "ip", Type: models.FieldTypeIP},
	})
	if err != nil {
		t.Fatalf("GenerateClientCode returned error: %v", err)
	}
	if !strings.Contains(code, "/api/report") {
		t.Errorf("generated code should contain /api/report path")
	}
}
