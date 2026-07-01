package controllers

import (
	"net/http"
	"testing"
)

func TestAdminServerServesAPI(t *testing.T) {
	ctx := setupTest(t)
	defer tearDown(t, ctx)
	resp, err := http.Get(ctx.adminServer.URL + "/api/campaigns/")
	if err != nil {
		t.Fatalf("error requesting api endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
	}
}
