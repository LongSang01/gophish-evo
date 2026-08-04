package models

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/url"
	"path"
	"strings"
	"sync"
	"text/template"

	log "github.com/gophish/gophish/logger"
	"github.com/skip2/go-qrcode"
)

// TemplateContext is an interface that allows both campaigns and email
// requests to have a PhishingTemplateContext generated for them.
type TemplateContext interface {
	getFromAddress() string
	getBaseURL() string
}

// PhishingTemplateContext is the context that is sent to any template, such
// as the email or landing page content.
type PhishingTemplateContext struct {
	From        string
	URL         string
	Tracker     string
	TrackingURL string
	RId         string
	BaseURL     string
	QRCode      string
	phishURL    string
	BaseRecipient
}

// FirstName returns the first name portion of FullName for template
// backward compatibility with {{.FirstName}}.
func (p PhishingTemplateContext) FirstName() string {
	parts := strings.SplitN(p.FullName, " ", 2)
	return parts[0]
}

// LastName returns the last name portion of FullName for template
// backward compatibility with {{.LastName}}.
func (p PhishingTemplateContext) LastName() string {
	parts := strings.SplitN(p.FullName, " ", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// NewPhishingTemplateContext returns a populated PhishingTemplateContext,
// parsing the correct fields from the provided TemplateContext and recipient.
func NewPhishingTemplateContext(ctx TemplateContext, r BaseRecipient, rid string) (PhishingTemplateContext, error) {
	f, err := mail.ParseAddress(ctx.getFromAddress())
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	fn := f.Name
	if fn == "" {
		fn = f.Address
	}
	templateURL, err := ExecuteTemplate(ctx.getBaseURL(), r)
	if err != nil {
		return PhishingTemplateContext{}, err
	}

	// For the base URL, we'll reset the the path and the query
	// This will create a URL in the form of http://example.com
	baseURL, err := url.Parse(templateURL)
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	baseURL.Path = ""
	baseURL.RawQuery = ""

	phishURL, _ := url.Parse(templateURL)
	q := phishURL.Query()
	q.Set(RecipientParameter, rid)
	phishURL.RawQuery = q.Encode()

	trackingURL, _ := url.Parse(templateURL)
	trackingURL.Path = path.Join(trackingURL.Path, "/track")
	trackingURL.RawQuery = q.Encode()

	// buildContext returns the template context with an empty QRCode field,
	// used as a fallback when QR generation fails.
	buildContext := func() PhishingTemplateContext {
		return PhishingTemplateContext{
			BaseRecipient: r,
			BaseURL:       baseURL.String(),
			URL:           phishURL.String(),
			TrackingURL:   trackingURL.String(),
			phishURL:      phishURL.String(),
			Tracker:       "<img alt='' style='display: none' src='" + trackingURL.String() + "'/>",
			From:          fn,
			RId:           rid,
			QRCode:        "",
		}
	}

	// Generate QR code for phishing URL
	qrCode, err := qrcode.New(phishURL.String(), qrcode.Medium)
	if err != nil {
		log.Warnf("QR code generation failed for %s: %v", phishURL.String(), err)
		return buildContext(), nil
	}
	pngBytes, err := qrCode.PNG(256)
	if err != nil {
		log.Warnf("QR code PNG encoding failed for %s: %v", phishURL.String(), err)
		return buildContext(), nil
	}
	base64QRCode := base64.StdEncoding.EncodeToString(pngBytes)
	imgSrc := fmt.Sprintf("data:image/png;base64,%s", base64QRCode)

	return PhishingTemplateContext{
		BaseRecipient: r,
		BaseURL:       baseURL.String(),
		URL:           phishURL.String(),
		TrackingURL:   trackingURL.String(),
		phishURL:      phishURL.String(),
		Tracker:       "<img alt='' style='display: none' src='" + trackingURL.String() + "'/>",
		From:          fn,
		RId:           rid,
		QRCode:        "<img alt='QRCode' src='" + imgSrc + "'/>",
	}, nil
}

// templateCache caches parsed templates keyed by the template text itself
// rather than by a template ID. Keying on content means edits to a template or
// page body immediately produce a new cache entry, so rendered output is
// always based on the current content. The parsed templates are read-only and
// safe for concurrent use.
var templateCache sync.Map

// maxTemplateCacheSize bounds the size of a template body that is kept in
// templateCache. Content larger than this (e.g. multi-MB attachment bodies such
// as Office documents or PDFs) is parsed on each call but never cached, so the
// cache can't grow without bound and leak memory in a long-running process.
const maxTemplateCacheSize = 64 * 1024

// ExecuteTemplate creates a templated string based on the provided
// template body and data.
func ExecuteTemplate(text string, data interface{}) (string, error) {
	var tmplIface interface{}
	var ok bool
	if len(text) <= maxTemplateCacheSize {
		tmplIface, ok = templateCache.Load(text)
	}
	if !ok {
		tmpl, err := template.New("template").Parse(text)
		if err != nil {
			return "", err
		}
		// Only cache entries within the size bound; very large bodies bypass
		// the cache entirely so they can't accumulate in memory.
		if len(text) <= maxTemplateCacheSize {
			tmplIface, _ = templateCache.LoadOrStore(text, tmpl)
		} else {
			tmplIface = tmpl
		}
	}
	buff := bytes.Buffer{}
	err := tmplIface.(*template.Template).Execute(&buff, data)
	return buff.String(), err
}

// ValidationContext is used for validating templates and pages
type ValidationContext struct {
	FromAddress string
	BaseURL     string
}

func (vc ValidationContext) getFromAddress() string {
	return vc.FromAddress
}

func (vc ValidationContext) getBaseURL() string {
	return vc.BaseURL
}

// ValidateTemplate ensures that the provided text in the page or template
// uses the supported template variables correctly.
func ValidateTemplate(text string) error {
	vc := ValidationContext{
		FromAddress: "foo@bar.com",
		BaseURL:     "http://example.com",
	}
	td := Result{
		BaseRecipient: BaseRecipient{
			Email:    "foo@bar.com",
			FullName: "Foo Bar",
			Position: "Test",
		},
		RId: "123456",
	}
	ptx, err := NewPhishingTemplateContext(vc, td.BaseRecipient, td.RId)
	if err != nil {
		return err
	}
	_, err = ExecuteTemplate(text, ptx)
	if err != nil {
		return err
	}
	return nil
}
