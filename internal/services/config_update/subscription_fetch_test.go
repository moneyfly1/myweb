package config_update

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchURLContentFallsBackToSubscriptionClientHeaders(t *testing.T) {
	const body = "anytls://password@example.com:443/?sni=example.com#example"
	var seenBrowserUA bool
	var seenV2rayNUA bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.UserAgent(), "Mozilla"):
			seenBrowserUA = true
			http.Error(w, "bad request", http.StatusBadRequest)
		case strings.Contains(strings.ToLower(r.UserAgent()), "v2rayn"):
			seenV2rayNUA = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		default:
			http.Error(w, "unsupported client", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	originalProfiles := subscriptionRequestProfiles
	subscriptionRequestProfiles = []subscriptionRequestProfile{
		{Name: "browser", UserAgent: "Mozilla/5.0", Accept: "*/*"},
		{Name: "v2rayN", UserAgent: "v2rayN/6.23", Accept: "*/*"},
	}
	defer func() { subscriptionRequestProfiles = originalProfiles }()

	service := &ConfigUpdateService{}
	client := &http.Client{Timeout: 2 * time.Second}
	got, err := service.fetchURLContent(client, server.URL)
	if err != nil {
		t.Fatalf("fetchURLContent returned error: %v", err)
	}
	if string(got) != body {
		t.Fatalf("unexpected body: %q", got)
	}
	if !seenBrowserUA || !seenV2rayNUA {
		t.Fatalf("expected browser fallback to v2rayN, seenBrowserUA=%v seenV2rayNUA=%v", seenBrowserUA, seenV2rayNUA)
	}
}
