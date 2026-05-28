package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadProjectEnv(t *testing.T) {
	dir := t.TempDir()
	body := `# a comment
A=alpha
  B  =  beta with space
C="quoted value"
D='single-quoted'

E=
=no_key
F=g=h
# trailing comment
`
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadProjectEnv(path)
	cases := map[string]string{
		"A": "alpha",
		"B": "beta with space",
		"C": "quoted value",
		"D": "single-quoted",
		"E": "",
		"F": "g=h", // SplitN keeps the rest of the value verbatim
	}
	for k, want := range cases {
		if got[k] != want {
			t.Errorf("LoadProjectEnv[%q] = %q, want %q", k, got[k], want)
		}
	}
	if _, ok := got[""]; ok {
		t.Errorf("LoadProjectEnv kept the empty-key line")
	}
}

func TestLoadProjectEnv_MissingFileIsEmpty(t *testing.T) {
	m := LoadProjectEnv(filepath.Join(t.TempDir(), "does-not-exist.env"))
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestStringPrecedence(t *testing.T) {
	t.Setenv("MUSU_AI_URL", "")
	v := viper.New()
	v.SetDefault("ai_url", "http://default/v1")
	projectEnv := map[string]string{}

	if got := String(v, projectEnv, "ai_url", "MUSU_AI_URL"); got != "http://default/v1" {
		t.Errorf("default tier: got %q, want default", got)
	}

	projectEnv["MUSU_AI_URL"] = "http://project-env/v1"
	if got := String(v, projectEnv, "ai_url", "MUSU_AI_URL"); got != "http://project-env/v1" {
		t.Errorf(".env tier should beat default: got %q", got)
	}

	t.Setenv("MUSU_AI_URL", "http://process-env/v1")
	if got := String(v, projectEnv, "ai_url", "MUSU_AI_URL"); got != "http://process-env/v1" {
		t.Errorf("process env should beat .env: got %q", got)
	}

	// Blank-after-trim process env must NOT shadow lower layers.
	t.Setenv("MUSU_AI_URL", "   ")
	if got := String(v, projectEnv, "ai_url", "MUSU_AI_URL"); got != "http://project-env/v1" {
		t.Errorf("blank process env should fall through: got %q", got)
	}
}

func TestIntPrecedence(t *testing.T) {
	v := viper.New()
	v.SetDefault("imap_port", 993)
	projectEnv := map[string]string{}

	if got := Int(v, projectEnv, "imap_port", "MUSU_IMAP_PORT"); got != 993 {
		t.Errorf("default tier: got %d", got)
	}

	projectEnv["MUSU_IMAP_PORT"] = "1430"
	if got := Int(v, projectEnv, "imap_port", "MUSU_IMAP_PORT"); got != 1430 {
		t.Errorf(".env tier: got %d", got)
	}

	t.Setenv("MUSU_IMAP_PORT", "2143")
	if got := Int(v, projectEnv, "imap_port", "MUSU_IMAP_PORT"); got != 2143 {
		t.Errorf("process env tier: got %d", got)
	}

	// A garbage value falls through instead of coercing to zero.
	t.Setenv("MUSU_IMAP_PORT", "not-a-number")
	if got := Int(v, projectEnv, "imap_port", "MUSU_IMAP_PORT"); got != 1430 {
		t.Errorf("garbage process env should fall through to .env: got %d", got)
	}
}

func TestStringHandlesNilViper(t *testing.T) {
	if got := String(nil, nil, "k", "K"); got != "" {
		t.Errorf("nil viper + no envs should return empty, got %q", got)
	}
	if got := Int(nil, nil, "k", "K"); got != 0 {
		t.Errorf("nil viper + no envs should return 0, got %d", got)
	}
}
