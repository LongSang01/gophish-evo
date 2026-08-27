package imap

import (
	"net/textproto"
	"testing"

	"github.com/jordan-wright/email"
)

// ---------------------------------------------------------------------------
// goPhishRegex
// ---------------------------------------------------------------------------

func TestGoPhishRegexStandard(t *testing.T) {
	cases := []struct {
		name string
		url  string
		rid  string
	}{
		{"standard", "http://example.com/?rid=AbC1234", "AbC1234"},
		{"tracking pixel", "http://example.com/track?rid=XyZ9876", "XyZ9876"},
		{"quoted-printable", "http://example.com/?rid=3DAbC1234", "AbC1234"},
		{"ATP encoded ?=", "http://example.com/%3Frid%3DAbC1234", "AbC1234"},
		{"ATP encoded =3D", "http://example.com/%3Frid%3D3DAbC1234", "AbC1234"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches := goPhishRegex.FindStringSubmatch(tc.url)
			if matches == nil {
				t.Fatalf("no match for %q", tc.url)
			}
			got := matches[len(matches)-1]
			if got != tc.rid {
				t.Errorf("got rid %q, want %q", got, tc.rid)
			}
		})
	}
}

func TestGoPhishRegexNoMatch(t *testing.T) {
	cases := []string{
		"http://example.com/",
		"http://example.com/?rid=",
		"http://example.com/?rid=short",   // 5 chars, too short
		"http://example.com/?rid=123456",  // 6 chars, too short
		"http://example.com/?rid=under_sc", // underscore not in [A-Za-z0-9]
		"no url here",
		"",
	}
	for _, s := range cases {
		if goPhishRegex.MatchString(s) {
			t.Errorf("unexpected match for %q", s)
		}
	}
}

// ---------------------------------------------------------------------------
// checkRIDs
// ---------------------------------------------------------------------------

func newEmail(text, html string) *email.Email {
	return &email.Email{
		Text: []byte(text),
		HTML: []byte(html),
	}
}

func TestCheckRIDsFromText(t *testing.T) {
	em := newEmail("Click http://phish.example.com/?rid=AAAAAAA now", "")
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if !rids["AAAAAAA"] {
		t.Error("rid AAAAAAA not found in text")
	}
}

func TestCheckRIDsFromHTML(t *testing.T) {
	em := newEmail("", `<a href="http://phish.example.com/?rid=BBBBBBB">link</a>`)
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if !rids["BBBBBBB"] {
		t.Error("rid BBBBBBB not found in html")
	}
}

func TestCheckRIDsMultiple(t *testing.T) {
	em := newEmail(
		"http://a.com/?rid=1111111 http://b.com/?rid=2222222",
		"http://c.com/?rid=3333333",
	)
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	for _, want := range []string{"1111111", "2222222", "3333333"} {
		if !rids[want] {
			t.Errorf("rid %s not found", want)
		}
	}
	if len(rids) != 3 {
		t.Errorf("expected 3 rids, got %d", len(rids))
	}
}

func TestCheckRIDSDedup(t *testing.T) {
	em := newEmail("http://a.com/?rid=XXXXXXX http://b.com/?rid=XXXXXXX", "")
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if len(rids) != 1 {
		t.Errorf("expected 1 unique rid, got %d", len(rids))
	}
}

func TestCheckRIDsEmpty(t *testing.T) {
	em := newEmail("nothing here", "<p>no links</p>")
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if len(rids) != 0 {
		t.Errorf("expected 0 rids, got %d", len(rids))
	}
}

func TestCheckRIDsQuotedPrintable(t *testing.T) {
	em := newEmail("http://a.com/?rid=3DAAAAAAA", "")
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if !rids["AAAAAAA"] {
		t.Error("quoted-printable rid not found")
	}
}

func TestCheckRIDsATPEncoded(t *testing.T) {
	em := newEmail("http://a.com/%3Frid%3DBBBBBBB", "")
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if !rids["BBBBBBB"] {
		t.Error("ATP-encoded rid not found")
	}
}

// ---------------------------------------------------------------------------
// matchEmail (including .eml attachment parsing)
// ---------------------------------------------------------------------------

func TestMatchEmailNoAttachments(t *testing.T) {
	em := newEmail("http://a.com/?rid=CCCCCCC", "")
	rids, err := matchEmail(em)
	if err != nil {
		t.Fatal(err)
	}
	if !rids["CCCCCCC"] {
		t.Error("rid CCCCCCC not found")
	}
}

func TestMatchEmailWithEMLAttachment(t *testing.T) {
	// Build a nested .eml attachment that contains a rid
	inner := &email.Email{
		Text: []byte("http://phish.example.com/?rid=DDDDDDD"),
	}
	emlBytes, err := inner.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	outer := &email.Email{
		Text:        []byte("Forwarded email"),
		Attachments: []*email.Attachment{},
	}
	outer.Attachments = append(outer.Attachments, &email.Attachment{
		Filename: "forwarded.eml",
		Content:  emlBytes,
		Header:   textproto.MIMEHeader{"Content-Type": {"message/rfc822"}},
	})
	rids, err := matchEmail(outer)
	if err != nil {
		t.Fatal(err)
	}
	if !rids["DDDDDDD"] {
		t.Error("rid DDDDDDD not found in .eml attachment")
	}
}

func TestMatchEmailAttachmentNoRid(t *testing.T) {
	inner := &email.Email{
		Text: []byte("No links here"),
	}
	emlBytes, err := inner.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	outer := &email.Email{
		Text: []byte("http://a.com/?rid=EEEEEEE"),
	}
	outer.Attachments = append(outer.Attachments, &email.Attachment{
		Filename: "forwarded.eml",
		Content:  emlBytes,
		Header:   textproto.MIMEHeader{"Content-Type": {"message/rfc822"}},
	})
	rids, err := matchEmail(outer)
	if err != nil {
		t.Fatal(err)
	}
	if !rids["EEEEEEE"] {
		t.Error("rid EEEEEEE not found")
	}
	if len(rids) != 1 {
		t.Errorf("expected 1 rid, got %d", len(rids))
	}
}

func TestMatchEmailNonEMLAttachmentIgnored(t *testing.T) {
	outer := &email.Email{
		Text: []byte("http://a.com/?rid=FFFFFFF"),
	}
	outer.Attachments = append(outer.Attachments, &email.Attachment{
		Filename: "image.png",
		Content:  []byte("fake png data"),
		Header:   textproto.MIMEHeader{"Content-Type": {"image/png"}},
	})
	rids, err := matchEmail(outer)
	if err != nil {
		t.Fatal(err)
	}
	if !rids["FFFFFFF"] {
		t.Error("rid FFFFFFF not found")
	}
	if len(rids) != 1 {
		t.Errorf("expected 1 rid, got %d", len(rids))
	}
}

func TestMatchEmailEmpty(t *testing.T) {
	em := &email.Email{
		Text: []byte("just a normal email"),
		HTML: []byte("<p>no links</p>"),
	}
	rids, err := matchEmail(em)
	if err != nil {
		t.Fatal(err)
	}
	if len(rids) != 0 {
		t.Errorf("expected 0 rids, got %d", len(rids))
	}
}

// ---------------------------------------------------------------------------
// NewMonitor
// ---------------------------------------------------------------------------

func TestNewMonitorNotNil(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("NewMonitor returned nil")
	}
}
