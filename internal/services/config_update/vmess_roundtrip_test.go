package config_update

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Regression test:
// - Subscription sources often break `vmess://` base64 payload into multiple lines.
// - We must preserve `host` during ParseNodeLink -> vmessToLink roundtrip.
func TestVMessHostRoundTrip_ParseAndConvert(t *testing.T) {
	// Cleaned one-line vmess from `terminals/8.txt` (port=20306).
	const vmessExample = `vmess:ew0KICAidiI6ICIyIiwNCiAgInBzIjoi56S65L6L6IqC54K5IiwNCiAgImFkZCI6ICJub2RlMS5leGFtcGxlLmNvbSIsDQogICJwb3J0IjogIjIwMzA2IiwNCiAgImlkIjogIjAwMDAwMDAwLTAwMDAtNDAwMC04MDAwLTAwMDAwMDAwMDAwMyIsDQogICJhaWQiOiAiMCIsDQogICJzY3kiOiAiYXV0byIsDQogICJuZXQiOiAidGNwIiwNCiAgInR5cGUiOiAiaHR0cCIsDQogICJob3N0IjogIm5vZGUxLmV4YW1wbGUuY29tIiwNCiAgInBhdGgiOiAiLyIsDQogICJ0bHMiOiAiIiwNCiAgInNuaSI6ICIiLA0KICAiYWxwbiI6ICIiLA0KICAiZnAiOiAiIiwNCiAgImluc2VjdXJlIjogIjAiDQp9`

	origHost, err := getVmessHostFromLink(t, vmessExample)
	if err != nil {
		t.Fatalf("failed to decode original vmess host: %v", err)
	}

	node, err := ParseNodeLink(vmessExample)
	if err != nil {
		t.Fatalf("ParseNodeLink failed: %v", err)
	}
	if node.Type != "vmess" {
		t.Fatalf("unexpected node type: %s", node.Type)
	}

	svc := &ConfigUpdateService{}
	outLink := svc.vmessToLink(node)
	outHost, err := getVmessHostFromLink(t, outLink)
	if err != nil {
		t.Fatalf("failed to decode converted vmess host: %v", err)
	}

	if outHost != origHost {
		t.Fatalf("vmess host mismatch: original=%q converted=%q", origHost, outHost)
	}
}

func TestVMessHostRoundTrip_ExtractAndConvert(t *testing.T) {
	const vmessExample = `vmess:ew0KICAidiI6ICIyIiwNCiAgInBzIjoi56S65L6L6IqC54K5IiwNCiAgImFkZCI6ICJub2RlMS5leGFtcGxlLmNvbSIsDQogICJwb3J0IjogIjIwMzA2IiwNCiAgImlkIjogIjAwMDAwMDAwLTAwMDAtNDAwMC04MDAwLTAwMDAwMDAwMDAwMyIsDQogICJhaWQiOiAiMCIsDQogICJzY3kiOiAiYXV0byIsDQogICJuZXQiOiAidGNwIiwNCiAgInR5cGUiOiAiaHR0cCIsDQogICJob3N0IjogIm5vZGUxLmV4YW1wbGUuY29tIiwNCiAgInBhdGgiOiAiLyIsDQogICJ0bHMiOiAiIiwNCiAgInNuaSI6ICIiLA0KICAiYWxwbiI6ICIiLA0KICAiZnAiOiAiIiwNCiAgImluc2VjdXJlIjogIjAiDQp9`

	svc := &ConfigUpdateService{}
	// Simulate subscription sources that wrap base64 payload into multiple lines/spaces.
	content := "prefix " + injectWhitespaceIntoVmessLink(vmessExample, 48) + " suffix"

	links := svc.extractNodeLinks(content)
	if len(links) == 0 {
		t.Fatalf("extractNodeLinks returned 0 links")
	}

	var extracted string
	for _, l := range links {
		if strings.HasPrefix(strings.TrimSpace(l), "vmess://") {
			extracted = l
			break
		}
	}
	if extracted == "" {
		t.Fatalf("no vmess:// link extracted from content; got %d links", len(links))
	}

	origHost, err := getVmessHostFromLink(t, vmessExample)
	if err != nil {
		t.Fatalf("failed to decode original vmess host: %v", err)
	}

	node, err := ParseNodeLink(extracted)
	if err != nil {
		t.Fatalf("ParseNodeLink failed on extracted link: %v", err)
	}

	outLink := svc.vmessToLink(node)
	outHost, err := getVmessHostFromLink(t, outLink)
	if err != nil {
		t.Fatalf("failed to decode converted vmess host: %v", err)
	}

	if outHost != origHost {
		t.Fatalf("vmess host mismatch after extract/convert: original=%q converted=%q", origHost, outHost)
	}
}

func injectWhitespaceIntoVmessLink(link string, every int) string {
	const prefix = "vmess://"
	if !strings.HasPrefix(link, prefix) || every <= 0 {
		return link
	}
	payload := strings.TrimPrefix(link, prefix)
	var b strings.Builder
	b.WriteString(prefix)
	for i := 0; i < len(payload); i++ {
		b.WriteByte(payload[i])
		if (i+1)%every == 0 && i != len(payload)-1 {
			b.WriteByte('\n')
			if (i+1)%(every*2) == 0 {
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}

func getVmessHostFromLink(t *testing.T, link string) (string, error) {
	t.Helper()
	encoded := strings.TrimPrefix(strings.TrimSpace(link), "vmess://")
	if encoded == "" {
		t.Fatalf("empty vmess payload")
	}

	decoded, err := DecodeBase64(encoded)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return "", err
	}

	host, ok := data["host"].(string)
	if !ok || host == "" {
		return "", fmt.Errorf("vmess json missing non-empty string field `host`")
	}
	return host, nil
}

