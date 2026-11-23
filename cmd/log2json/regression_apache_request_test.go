package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegression_ApacheLinesWithoutAWellFormedRequest(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "request timeout logs a dash",
			line: `10.0.0.1 - - [15/Jan/2024:10:30:45 +0000] "-" 408 - "-" "-"`,
			want: `{"ip":"10.0.0.1","status":408,"timestamp":"15/Jan/2024:10:30:45 +0000"}`,
		},
		{
			name: "binary probe on the HTTP port",
			line: `203.0.113.9 - - [15/Jan/2024:10:30:46 +0000] "\x16\x03\x01" 400 226 "-" "-"`,
			want: `{"ip":"203.0.113.9","request":"\\x16\\x03\\x01","size":226,"status":400,"timestamp":"15/Jan/2024:10:30:46 +0000"}`,
		},
		{
			name: "method and path without protocol",
			line: `10.0.0.2 - - [15/Jan/2024:10:30:47 +0000] "GET /" 400 0`,
			want: `{"ip":"10.0.0.2","request":"GET /","size":0,"status":400,"timestamp":"15/Jan/2024:10:30:47 +0000"}`,
		},
		{
			name: "well-formed request is unchanged",
			line: `192.168.1.1 - bob [15/Jan/2024:10:30:45 +0000] "GET /page HTTP/1.1" 200 1234 "http://ref.com" "Mozilla/5.0"`,
			want: `{"ip":"192.168.1.1","method":"GET","path":"/page","protocol":"HTTP/1.1","referer":"http://ref.com","size":1234,"status":200,"timestamp":"15/Jan/2024:10:30:45 +0000","user":"bob","useragent":"Mozilla/5.0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cfg := Config{Format: "apache"}
			if err := runPipeline(cfg, strings.NewReader(tt.line+"\n"), &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(stdout.String()); got != tt.want {
				t.Errorf("\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestRegression_ApacheStreamKeepsParsingAfterATimeoutLine(t *testing.T) {
	input := `192.168.1.1 - - [15/Jan/2024:10:30:45 +0000] "GET /a HTTP/1.1" 200 10` + "\n" +
		`10.0.0.1 - - [15/Jan/2024:10:30:46 +0000] "-" 408 - "-" "-"` + "\n"

	var stdout, stderr bytes.Buffer
	if err := runPipeline(Config{}, strings.NewReader(input), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "_parseError") {
		t.Errorf("auto-detected apache stream reported a parse error:\n%s", stdout.String())
	}
}
