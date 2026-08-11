package models

import (
	"bytes"
	"container/list"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/url"
	"path"
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
// always based on the current content. The cache is bounded: both the maximum
// size of a single entry and the maximum number of entries are capped, so it
// can't grow without bound in a long-running process. Evicted entries are
// simply re-parsed on the next use, which is safe since templates are
// read-only.
type templateCache struct {
	mu      sync.Mutex
	ll      *list.List
	entries map[string]*list.Element
}

type templateCacheEntry struct {
	text string
	tmpl *template.Template
}

// maxTemplateCacheSize bounds the size of a single template body kept in the
// cache. Content larger than this (e.g. multi-MB attachment bodies such as
// Office documents or PDFs) is parsed on each call but never cached, so one
// huge attachment can't monopolize memory.
const maxTemplateCacheSize = 64 * 1024

// maxTemplateCacheEntries bounds the total number of cached templates.
// Least-recently-used entries are evicted once the cache is full, preventing
// unbounded growth from accumulated template and page versions.
const maxTemplateCacheEntries = 1024

var templateCacheInstance = &templateCache{
	ll:      list.New(),
	entries: make(map[string]*list.Element),
}

// load returns the cached template for text, marking it most-recently-used.
func (c *templateCache) load(text string) (*template.Template, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[text]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*templateCacheEntry).tmpl, true
}

// store inserts a template for text, evicting the least-recently-used entry if
// the cache is full. If text is already present it is just refreshed.
func (c *templateCache) store(text string, tmpl *template.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[text]; ok {
		c.ll.MoveToFront(el)
		return
	}
	if c.ll.Len() >= maxTemplateCacheEntries {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.entries, oldest.Value.(*templateCacheEntry).text)
		}
	}
	el := c.ll.PushFront(&templateCacheEntry{text: text, tmpl: tmpl})
	c.entries[text] = el
}

// ExecuteTemplate creates a templated string based on the provided
// template body and data.
func ExecuteTemplate(text string, data interface{}) (string, error) {
	var tmpl *template.Template
	var ok bool
	if len(text) <= maxTemplateCacheSize {
		tmpl, ok = templateCacheInstance.load(text)
	}
	if !ok {
		var err error
		tmpl, err = template.New("template").Parse(text)
		if err != nil {
			return "", err
		}
		// Only cache entries within the size bound; very large bodies bypass
		// the cache entirely so they can't monopolize memory.
		if len(text) <= maxTemplateCacheSize {
			templateCacheInstance.store(text, tmpl)
		}
	}
	buff := bytes.Buffer{}
	err := tmpl.Execute(&buff, data)
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
