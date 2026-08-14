package util

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWriteCSVBasic(t *testing.T) {
	rows := []CSVRow{
		{Fixed: []interface{}{int64(1), "1.1.1.1", time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)}},
		{Fixed: []interface{}{int64(2), "2.2.2.2", time.Date(2026, 8, 14, 9, 30, 0, 0, time.UTC)}},
	}
	var buf bytes.Buffer
	if err := WriteCSV(&buf, []string{"id", "ip", "created_at"}, rows); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"id,ip,created_at", "1,1.1.1.1,2026-08-14T08:00:00Z", "2,2.2.2.2,2026-08-14T09:30:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
	if !strings.HasPrefix(got, "\uFEFF") {
		t.Errorf("expected UTF-8 BOM prefix, got:\n%s", got)
	}
}

func TestWriteCSVDedupCollision(t *testing.T) {
	rows := []CSVRow{
		{
			Fixed: []interface{}{int64(1), "203.0.113.9"},
			Data:  map[string]interface{}{"ip": "eth0:192.168.1.100", "mac": "00:11:22:33:44:55", "username": "zhangsan"},
		},
		{
			Fixed: []interface{}{int64(2), "203.0.113.10"},
			Data:  map[string]interface{}{"ip": "eth0:192.168.1.101", "mac": "00:11:22:33:44:66"},
		},
	}
	var buf bytes.Buffer
	if err := WriteCSV(&buf, []string{"id", "ip"}, rows); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}
	got := buf.String()
	for _, want := range []string{
		"id,ip,ip1,mac,username", // reported ip de-duplicated to ip1
		"eth0:192.168.1.100",
		"eth0:192.168.1.101",
		"203.0.113.9",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
}

func TestWriteCSVRepeatedCollision(t *testing.T) {
	rows := []CSVRow{
		{
			Fixed: []interface{}{int64(1), time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)},
			Data:  map[string]interface{}{"created_at": "custom", "id": "a", "id1": "b"},
		},
	}
	var buf bytes.Buffer
	if err := WriteCSV(&buf, []string{"id", "created_at"}, rows); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}
	got := buf.String()
	for _, want := range []string{
		"id,created_at,created_at1,id1,id11", // each collision gets its own suffixed column
		"1",
		"2026-08-14T08:00:00Z",
		"custom",
		"a",
		"b",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
}

func TestSanitizeCSVValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal text", "hello", "hello"},
		{"empty string", "", ""},
		{"formula equals", "=cmd", "'=cmd"},
		{"formula plus", "+SUM(A1)", "'+SUM(A1)"},
		{"formula minus", "-2+3", "'-2+3"},
		{"formula at", "@SUM", "'@SUM"},
		{"formula tab", "\tdata", "'\tdata"},
		{"formula cr", "\rdata", "'\rdata"},
		{"safe dash in middle", "a-b", "a-b"},
		{"safe number", "123", "123"},
		{"space prefix", " hello", " hello"},
		{"single char equals", "=", "'="},
		{"single char plus", "+", "'+"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCSVValue(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeCSVValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteCSVFormulaInjection(t *testing.T) {
	rows := []CSVRow{
		{Fixed: []interface{}{int64(1), "=SUM(A1:A2)", "+2"}},
	}
	var buf bytes.Buffer
	if err := WriteCSV(&buf, []string{"id", "ip", "note"}, rows); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "'=SUM(A1:A2)") || !strings.Contains(got, "'+2") {
		t.Errorf("formula values not escaped, got:\n%s", got)
	}
}
