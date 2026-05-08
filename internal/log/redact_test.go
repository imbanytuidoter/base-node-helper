package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRedactRPCURLWithAPIKey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"alchemy mainnet",
			"https://eth-mainnet.g.alchemy.com/v2/AAAABBBBCCCCDDDDEEEEFFFF12345678",
			"https://eth-mainnet.g.alchemy.com/v2/****",
		},
		{
			"infura",
			"https://mainnet.infura.io/v3/0123456789abcdef0123456789abcdef",
			"https://mainnet.infura.io/v3/****",
		},
		{
			"quicknode",
			"https://example.quiknode.pro/abcdef1234567890abcdef1234567890/",
			"https://example.quiknode.pro/****/",
		},
		{
			"plain http no key untouched",
			"https://reth-archive:8545",
			"https://reth-archive:8545",
		},
		{
			"bearer token in line",
			"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.AAAA",
			"Authorization: Bearer ****",
		},
		{
			// 9-char path component must not be redacted (false-positive guard)
			"path component not redacted",
			"https://eth-mainnet.g.alchemy.com/v3/contracts",
			"https://eth-mainnet.g.alchemy.com/v3/contracts",
		},
		{
			"basic auth credentials in URL",
			"https://user:mysupersecretpassword123@eth-mainnet.example.com/rpc",
			"https://****:****@eth-mainnet.example.com/rpc",
		},
		{
			"websocket RPC URL with API key",
			"wss://eth-mainnet.g.alchemy.com/v2/AAAABBBBCCCCDDDDEEEEFFFF12345678",
			"wss://eth-mainnet.g.alchemy.com/v2/****",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Redact(c.in)
			if got != c.want {
				t.Fatalf("Redact(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactEmptyString(t *testing.T) {
	if got := Redact(""); got != "" {
		t.Fatalf("Redact(\"\") = %q, want \"\"", got)
	}
}

func TestRedactIdempotent(t *testing.T) {
	inputs := []string{
		"https://eth-mainnet.g.alchemy.com/v2/AAAABBBBCCCCDDDDEEEEFFFF12345678",
		"https://mainnet.infura.io/v3/0123456789abcdef0123456789abcdef",
		"https://example.quiknode.pro/abcdef1234567890abcdef1234567890/",
		"https://reth-archive:8545",
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.AAAA",
		"",
	}
	for _, s := range inputs {
		once := Redact(s)
		twice := Redact(once)
		if once != twice {
			t.Fatalf("Redact not idempotent for %q: first=%q second=%q", s, once, twice)
		}
	}
}

func TestRedactPreservesJSON(t *testing.T) {
	var buf bytes.Buffer
	lg := New(&buf, zerolog.InfoLevel)
	lg.Info().Str("rpc_url", "https://eth-mainnet.g.alchemy.com/v2/AAAABBBBCCCCDDDDEEEEFFFF12345678").Msg("test")

	raw := buf.String()

	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("output is not valid JSON after redaction: %v\noutput: %s", err, raw)
	}

	rpcURL, ok := out["rpc_url"].(string)
	if !ok {
		t.Fatalf("rpc_url field missing or not a string in: %s", raw)
	}
	if !strings.HasSuffix(rpcURL, "****") {
		t.Fatalf("rpc_url not redacted in JSON output: %s", rpcURL)
	}
}

func TestRedactMultipleSecretsPerLine(t *testing.T) {
	in := "rpc=https://eth-mainnet.g.alchemy.com/v2/AAAABBBBCCCCDDDDEEEEFFFF12345678 Authorization: Bearer SuperSecretBearerToken123456789"
	got := Redact(in)

	if strings.Contains(got, "AAAABBBBCCCCDDDDEEEEFFFF12345678") {
		t.Fatalf("v2 key not redacted: %s", got)
	}
	if strings.Contains(got, "SuperSecretBearerToken123456789") {
		t.Fatalf("bearer token not redacted: %s", got)
	}
	// both placeholders must appear
	if strings.Count(got, "****") < 2 {
		t.Fatalf("expected at least 2 redaction markers, got: %s", got)
	}
}

func TestNewNilWriterFallsBackToStderr(t *testing.T) {
	// nil w must not panic; it falls back to os.Stderr internally.
	lg := New(nil, zerolog.InfoLevel)
	_ = lg
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRedactedWriterPropagatesError(t *testing.T) {
	w := redactedWriter{errWriter{}}
	_, err := w.Write([]byte("any"))
	if err == nil {
		t.Fatal("expected Write to return an error, got nil")
	}
}

func TestLoggerRedactsFields(t *testing.T) {
	var buf bytes.Buffer
	lg := New(&buf, zerolog.InfoLevel)
	lg.Info().Str("rpc_url", "https://eth-mainnet.g.alchemy.com/v2/AAAABBBBCCCCDDDDEEEEFFFF12345678").Msg("starting")
	out := buf.String()
	if strings.Contains(out, "AAAABBBBCCCCDDDDEEEEFFFF12345678") {
		t.Fatalf("logger leaked secret: %s", out)
	}
	if !strings.Contains(out, "****") {
		t.Fatalf("logger did not redact: %s", out)
	}
}
