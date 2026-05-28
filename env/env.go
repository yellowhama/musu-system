// Package env merges three configuration layers for a project-scoped tool:
//
//  1. process environment (e.g. NURIKUN_AI_URL=...) — highest precedence
//  2. project-local .env file (KEY=VAL lines) — middle precedence
//  3. viper defaults / config.yaml — lowest precedence
//
// It is the shared environment loader used by every CLI in the Musu ecosystem
// (crawl-ai, marketer, nurikun) so secrets stay out of git and the precedence
// rules are identical across tools.
package env

import (
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// LoadProjectEnv parses a simple `KEY=VAL` file. Lines beginning with `#` are
// treated as comments. Surrounding single or double quotes are stripped from
// values. A missing file returns an empty map (not an error) so callers can
// treat the project .env as optional.
func LoadProjectEnv(path string) map[string]string {
	values := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	}
	return values
}

// String returns the first non-empty value of process env[envKey], the
// project-local .env map[envKey], and the viper key (in that precedence).
// Empty-after-trim values are ignored so a blank env var does not shadow a
// real config.yaml value.
func String(v *viper.Viper, projectEnv map[string]string, key, envKey string) string {
	if value, ok := os.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if value, ok := projectEnv[envKey]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	if v == nil {
		return ""
	}
	return v.GetString(key)
}

// Int is the integer-typed sibling of String. A non-parseable env value falls
// through to the next layer instead of erroring, so a typo in a .env does not
// silently coerce to zero — the viper default still wins.
func Int(v *viper.Viper, projectEnv map[string]string, key, envKey string) int {
	if value, ok := os.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	if value, ok := projectEnv[envKey]; ok && strings.TrimSpace(value) != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	if v == nil {
		return 0
	}
	return v.GetInt(key)
}
