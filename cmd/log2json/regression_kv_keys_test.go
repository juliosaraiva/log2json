package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegression_KeyValueKeysKeepDotsAndDashes(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "dotted and dashed keys",
			line: `user.id=42 request-id=abc http.status=200`,
			want: `{"http.status":200,"request-id":"abc","user.id":42}`,
		},
		{
			name: "keys that used to collide after truncation",
			line: `src.port=443 dst.port=51000`,
			want: `{"dst.port":51000,"src.port":443}`,
		},
		{
			name: "plain keys are unchanged",
			line: `level=info msg="user logged in" user_id=123`,
			want: `{"level":"info","msg":"user logged in","user_id":123}`,
		},
		{
			name: "leading punctuation is not part of a key",
			line: `--verbose=true -x=1 .hidden=2`,
			want: `{"hidden":2,"verbose":true,"x":1}`,
		},
	}

	for _, cfg := range []Config{{}, {Format: "kv"}} {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if err := runPipeline(cfg, strings.NewReader(tt.line+"\n"), &stdout, &stderr); err != nil {
					t.Fatal(err)
				}
				if got := strings.TrimSpace(stdout.String()); got != tt.want {
					t.Errorf("\n got: %s\nwant: %s", got, tt.want)
				}
			})
		}
	}
}
