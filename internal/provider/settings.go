// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// The environment variables and configuration file the Kaiten CLI reads. The
// provider reads the same ones, so a machine where `kaiten` works needs no
// further setup for Terraform.
const (
	envBaseURL   = "KAITEN_BASE_URL"
	envAuthToken = "KAITEN_AUTH_TOKEN"
)

// Sources a setting can come from, highest precedence first.
const (
	sourceConfig = "provider configuration"
	sourceEnv    = "environment"
	sourceFile   = "Kaiten CLI configuration file"
)

// settings are the connection settings the provider resolved, and where each
// came from. The token is never printed: String redacts it.
type settings struct {
	BaseURL         string
	BaseURLSource   string
	AuthToken       string
	AuthTokenSource string
}

func (s settings) String() string {
	return fmt.Sprintf("{BaseURL:%s (%s) AuthToken:<redacted> (%s)}", s.BaseURL, s.BaseURLSource, s.AuthTokenSource)
}

func (s settings) GoString() string { return s.String() }

// cliConfig is the subset of the Kaiten CLI configuration file the provider
// reads. The CLI writes it with `kaiten config set`.
type cliConfig struct {
	BaseURL   string `yaml:"base_url"`
	AuthToken string `yaml:"auth_token"`
}

// resolver resolves the provider's connection settings from, in order, the
// provider block, the environment and the Kaiten CLI configuration file. Each
// setting is resolved on its own: there is one secret, so nothing can be mixed
// across sources the way a key and a secret could.
type resolver struct {
	getenv func(string) string
	// cliConfigPath returns where the Kaiten CLI keeps its configuration file.
	cliConfigPath func() (string, error)
}

func newResolver() resolver {
	return resolver{getenv: os.Getenv, cliConfigPath: defaultCLIConfigPath}
}

// defaultCLIConfigPath mirrors the CLI's own config.DefaultPath.
func defaultCLIConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kaiten", "config.yaml"), nil
}

// resolve returns the resolved settings and any warnings worth surfacing. It
// fails when a source is present but unusable, or when a setting has no source.
func (r resolver) resolve(configBaseURL, configAuthToken string) (settings, []string, error) {
	file, path, warnings, err := r.loadCLIConfig()
	if err != nil {
		return settings{}, nil, err
	}

	var s settings
	s.BaseURL, s.BaseURLSource = firstSet(
		sourced{sourceConfig, configBaseURL},
		sourced{sourceEnv, r.getenv(envBaseURL)},
		sourced{sourceFile, file.BaseURL},
	)
	s.AuthToken, s.AuthTokenSource = firstSet(
		sourced{sourceConfig, configAuthToken},
		sourced{sourceEnv, r.getenv(envAuthToken)},
		sourced{sourceFile, file.AuthToken},
	)

	var missing []error
	if s.BaseURL == "" {
		missing = append(missing, missingError("base_url", envBaseURL, "base_url", path))
	}
	if s.AuthToken == "" {
		missing = append(missing, missingError("auth_token", envAuthToken, "auth_token", path))
	}
	if len(missing) > 0 {
		return settings{}, warnings, errors.Join(missing...)
	}

	return s, warnings, nil
}

// loadCLIConfig reads the Kaiten CLI configuration file. A file that does not
// exist offers nothing and is not an error; one that exists but cannot be read
// or parsed is, so a broken file is never silently skipped.
func (r resolver) loadCLIConfig() (cliConfig, string, []string, error) {
	path, err := r.cliConfigPath()
	if err != nil {
		// No config directory on this machine: there is no file to read.
		return cliConfig{}, "", nil, nil //nolint:nilerr // an unresolvable config dir means no CLI file, not a failure
	}

	data, err := os.ReadFile(path) //nolint:gosec // the path is the CLI's fixed location, not user input
	if errors.Is(err, fs.ErrNotExist) {
		return cliConfig{}, path, nil, nil
	}
	if err != nil {
		return cliConfig{}, path, nil, fmt.Errorf("reading the Kaiten CLI configuration file %s: %w", path, err)
	}

	var cfg cliConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cliConfig{}, path, nil, fmt.Errorf("parsing the Kaiten CLI configuration file %s: %w", path, err)
	}

	var warnings []string
	if info, err := os.Stat(path); err == nil && runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		warnings = append(warnings, fmt.Sprintf(
			"The Kaiten CLI configuration file %s holds an API token and is readable by other users. Restrict it with: chmod 0600 %s",
			path, path,
		))
	}

	return cfg, path, warnings, nil
}

type sourced struct {
	source string
	value  string
}

// firstSet returns the first non-blank value and the source it came from.
func firstSet(candidates ...sourced) (string, string) {
	for _, c := range candidates {
		if v := strings.TrimSpace(c.value); v != "" {
			return v, c.source
		}
	}
	return "", ""
}

func missingError(attribute, env, fileKey, path string) error {
	where := "the Kaiten CLI configuration file"
	if path != "" {
		where += " (" + path + ")"
	}
	return fmt.Errorf("%s is not set. Sources tried, in order: the provider's %s attribute, the %s environment variable, and the %s key of %s",
		attribute, attribute, env, fileKey, where)
}
