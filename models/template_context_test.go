package models

import (
	"encoding/base64"
	"fmt"
	"net/url"

	qrcode "github.com/skip2/go-qrcode"

	check "gopkg.in/check.v1"
)

type mockTemplateContext struct {
	URL         string
	FromAddress string
}

func (m mockTemplateContext) getFromAddress() string {
	return m.FromAddress
}

func (m mockTemplateContext) getBaseURL() string {
	return m.URL
}

func (s *ModelsSuite) TestNewTemplateContext(c *check.C) {
	r := Result{
		BaseRecipient: BaseRecipient{
			FullName: "Foo Bar",
			Email:    "foo@bar.com",
		},
		RId: "1234567",
	}
	ctx := mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From Address <from@example.com>",
	}
	phishURL, _ := url.Parse(fmt.Sprintf("%s?rid=%s", ctx.URL, r.RId))
	expected := PhishingTemplateContext{
		URL:           phishURL.String(),
		BaseURL:       ctx.URL,
		BaseRecipient: r.BaseRecipient,
		TrackingURL:   fmt.Sprintf("%s/track?rid=%s", ctx.URL, r.RId),
		From:          "From Address",
		RId:           r.RId,
		phishURL:      phishURL.String(),
	}
	expected.Tracker = "<img alt='' style='display: none' src='" + expected.TrackingURL + "'/>"
	qrCode, err := qrcode.New(phishURL.String(), qrcode.Medium)
	c.Assert(err, check.Equals, nil)
	pngBytes, err := qrCode.PNG(256)
	c.Assert(err, check.Equals, nil)
	base64QRCode := base64.StdEncoding.EncodeToString(pngBytes)
	expected.QRCode = "<img alt='QRCode' src='data:image/png;base64," + base64QRCode + "'/>"

	got, err := NewPhishingTemplateContext(ctx, r.BaseRecipient, r.RId)
	c.Assert(err, check.Equals, nil)
	c.Assert(got, check.DeepEquals, expected)
}
