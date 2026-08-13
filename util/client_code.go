package util

import (
	"bytes"
	"text/template"

	"github.com/gophish/gophish/models"
)

// ClientCodeParams are the values injected into the generated Go source.
type ClientCodeParams struct {
	BaseURL    string
	ReportKey  string
	CampaignID int64
	DedupKey   string
	Fields     []models.ReportField
}

// clientCodeTmpl is a complete, self-contained Go program that collects the
// configured machine fields and reports them to the configured endpoint.
// It intentionally uses only the standard library so the operator can
// compile it on any platform with a plain `go build`.
var clientCodeTmpl = `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"{{if needsOSUser}}
	"os/user"{{end}}
	"strings"
	"time"
)

const (
	reportURL  = "{{.BaseURL}}/api/report"
	reportKey  = "{{.ReportKey}}"
	campaignID = int64({{.CampaignID}})
)

func collectField(key string) string {
	switch key {
{{- range .Fields}}
	case "{{.Key}}":
{{- if eq .Type "ip"}}
		return nonLoopbackIPs()
{{- else if eq .Type "mac"}}
		return nonLoopbackMACs()
{{- else if eq .Type "username"}}
		if u, err := user.Current(); err == nil { return u.Username }
{{- else if eq .Type "hostname"}}
		if h, err := os.Hostname(); err == nil { return h }
{{- else}}
		// TODO: 自定义字段 "{{.Key}}" ({{.Type}})，请实现采集逻辑
		return ""
{{- end}}
{{- end}}
	}
	return ""
}

func nonLoopbackIPs() string {
	ifaces, _ := net.Interfaces()
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 { continue }
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
				ips = append(ips, iface.Name+":"+n.IP.String())
			}
		}
	}
	return strings.Join(ips, ",")
}

func nonLoopbackMACs() string {
	ifaces, _ := net.Interfaces()
	var macs []string
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) > 0 && iface.Flags&net.FlagLoopback == 0 {
			macs = append(macs, iface.Name+":"+iface.HardwareAddr.String())
		}
	}
	return strings.Join(macs, ",")
}

func main() {
	data := map[string]interface{}{}
{{range .Fields}}	data["{{.Key}}"] = collectField("{{.Key}}")
{{end}}
	payload, err := json.Marshal(map[string]interface{}{
		"campaign_id": campaignID, "source": "client", "data": data,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal failed:", err)
		os.Exit(1)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", reportURL, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Report-Key", reportKey)
		resp, err := client.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				fmt.Println("report success"); return
			}
			fmt.Fprintf(os.Stderr, "status %d, ", resp.StatusCode)
		}
		fmt.Fprintf(os.Stderr, "retry %d/5: %v\n", i+1, err)
		time.Sleep(5 * time.Second)
	}
	fmt.Fprintln(os.Stderr, "report failed after 5 retries")
	os.Exit(1)
}
`

// GenerateClientCode renders the standalone Go client source for a client-type
// campaign.
func GenerateClientCode(baseURL, reportKey, dedupKey string, campaignID int64, fields []models.ReportField) (string, error) {
	tmpl, err := template.New("client").Funcs(template.FuncMap{
		"needsOSUser": func() bool {
			for _, f := range fields {
				if f.Type == models.FieldTypeUsername {
					return true
				}
			}
			return false
		},
	}).Parse(clientCodeTmpl)
	if err != nil {
		return "", err
	}
	params := ClientCodeParams{
		BaseURL:    baseURL,
		ReportKey:  reportKey,
		DedupKey:   dedupKey,
		CampaignID: campaignID,
		Fields:     fields,
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}
