// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testResolver builds a resolver over a fake environment and a CLI
// configuration file holding content, or no file when content is empty.
func testResolver(t *testing.T, env map[string]string, content string) resolver {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return resolver{
		getenv:        func(key string) string { return env[key] },
		cliConfigPath: func() (string, error) { return path, nil },
	}
}

func TestResolverPrecedence(t *testing.T) {
	const file = "base_url: https://file.example/api\nauth_token: ksh_file\n"
	env := map[string]string{envBaseURL: "https://env.example/api", envAuthToken: "ksh_env"}

	tests := []struct {
		name                       string
		env                        map[string]string
		file                       string
		configURL, configToken     string
		wantURL, wantToken, source string
	}{
		{"config beats env and file", env, file, "https://config.example/api", "ksh_config", "https://config.example/api", "ksh_config", sourceConfig},
		{"env beats file", env, file, "", "", "https://env.example/api", "ksh_env", sourceEnv},
		{"file when nothing else is set", nil, file, "", "", "https://file.example/api", "ksh_file", sourceFile},
		{"blank values fall through", map[string]string{envAuthToken: "  "}, file, " ", "", "https://file.example/api", "ksh_file", sourceFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := testResolver(t, tt.env, tt.file).resolve(tt.configURL, tt.configToken)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got.BaseURL != tt.wantURL || got.AuthToken != tt.wantToken {
				t.Errorf("got %q/%q, want %q/%q", got.BaseURL, got.AuthToken, tt.wantURL, tt.wantToken)
			}
			if got.AuthTokenSource != tt.source {
				t.Errorf("token source = %q, want %q", got.AuthTokenSource, tt.source)
			}
		})
	}
}

func TestResolverSettingsResolveIndependently(t *testing.T) {
	r := testResolver(t, map[string]string{envAuthToken: "ksh_env"}, "base_url: https://file.example/api\n")

	got, _, err := r.resolve("", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.BaseURLSource != sourceFile || got.AuthTokenSource != sourceEnv {
		t.Errorf("sources = %q/%q, want %q/%q", got.BaseURLSource, got.AuthTokenSource, sourceFile, sourceEnv)
	}
}

func TestResolverMissingNamesEverySource(t *testing.T) {
	_, _, err := testResolver(t, nil, "").resolve("", "")
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"base_url", "auth_token", envBaseURL, envAuthToken, "config.yaml"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestResolverMalformedFileFails(t *testing.T) {
	r := testResolver(t, map[string]string{envBaseURL: "https://env.example/api", envAuthToken: "ksh_env"}, "base_url: [unterminated\n")

	if _, _, err := r.resolve("", ""); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Fatalf("expected a parse error, got %v", err)
	}
}

func TestResolverWarnsOnReadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits do not apply on Windows")
	}

	r := testResolver(t, nil, "base_url: https://file.example/api\nauth_token: ksh_file\n")
	path, _ := r.cliConfigPath()
	if err := os.Chmod(path, 0o644); err != nil { //nolint:gosec // the point is a file others can read
		t.Fatal(err)
	}

	_, warnings, err := r.resolve("", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "chmod 0600") {
		t.Errorf("warnings = %q, want one chmod warning", warnings)
	}
}

func TestSettingsRedactToken(t *testing.T) {
	s := settings{BaseURL: "https://kaiten.example/api", AuthToken: "ksh_secret"}

	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		if out := fmt.Sprintf(format, s); strings.Contains(out, "ksh_secret") {
			t.Errorf("%s leaks the token: %s", format, out)
		}
	}
}
