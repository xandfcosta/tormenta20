package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeEnvFile drops content in a temp dir and returns its path.
func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env.test")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

// sandboxEnv registers each key with t.Setenv so the test's own writes AND the
// ones loadEnvFile makes are restored afterwards — os.Setenv alone would leak
// into every test that runs later in this package.
func sandboxEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestLoadEnvFileExportsWhatTheProcessDoesNotSet(t *testing.T) {
	sandboxEnv(t, "T20_PORT")
	path := writeEnvFile(t, "T20_PORT=3001\n")

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile: %v", err)
	}

	if got := os.Getenv("T20_PORT"); got != "3001" {
		t.Errorf("T20_PORT = %q, want %q", got, "3001")
	}
}

// The process wins so `PORT=4000 pnpm start` overrides the environment's file
// without editing it — and so a runner that exports everything needs no file.
func TestLoadEnvFileKeepsTheValueTheProcessAlreadyExported(t *testing.T) {
	t.Setenv("T20_PORT", "4000")
	path := writeEnvFile(t, "T20_PORT=3001\n")

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile: %v", err)
	}

	if got := os.Getenv("T20_PORT"); got != "4000" {
		t.Errorf("T20_PORT = %q, want the exported %q", got, "4000")
	}
}

// A missing file is the CI/systemd shape: everything is exported already.
func TestLoadEnvFileTreatsAMissingFileAsNoConfiguration(t *testing.T) {
	if err := loadEnvFile(filepath.Join(t.TempDir(), "absent")); err != nil {
		t.Fatalf("missing file must not fail the boot: %v", err)
	}
}

// A skipped typo means a default the operator believes they changed, so the
// boot fails pointing at the line.
func TestLoadEnvFileRejectsALineThatIsNotKeyValue(t *testing.T) {
	sandboxEnv(t, "T20_PORT")
	path := writeEnvFile(t, "T20_PORT=3001\nJWT_SECRET\n")

	err := loadEnvFile(path)

	if err == nil {
		t.Fatal("expected an error for a line without '='")
	}
	if !strings.Contains(err.Error(), ":2:") || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("error must name the line and its content, got %q", err)
	}
}

func TestParseEnvLine(t *testing.T) {
	cases := []struct {
		name, raw, key, value string
		wantErr               bool
	}{
		{name: "plain", raw: "PORT=3001", key: "PORT", value: "3001"},
		{name: "blank", raw: "   "},
		{name: "comment", raw: "# PORT=3001"},
		{name: "spaces around the pair", raw: "  PORT = 3001  ", key: "PORT", value: "3001"},
		{name: "empty value", raw: "STATIC_DIR=", key: "STATIC_DIR"},
		// The DSN and a base64 secret both carry '='; only the first one splits.
		{name: "value with =", raw: "DATABASE_URL=file:./t20.db?a=b", key: "DATABASE_URL", value: "file:./t20.db?a=b"},
		{name: "double quoted", raw: `CORS_ORIGIN="http://localhost:5173"`, key: "CORS_ORIGIN", value: "http://localhost:5173"},
		{name: "single quoted keeps spaces", raw: "NAME=' a '", key: "NAME", value: " a "},
		// No inline comments: a '#' is part of the value, so a secret survives it.
		{name: "hash inside the value", raw: "JWT_SECRET=ab#cd", key: "JWT_SECRET", value: "ab#cd"},
		{name: "no equals", raw: "JWT_SECRET", wantErr: true},
		{name: "no key", raw: "=3001", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, value, err := parseEnvLine(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseEnvLine(%q) = (%q, %q, nil), want an error", tc.raw, key, value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEnvLine(%q): %v", tc.raw, err)
			}
			if key != tc.key || value != tc.value {
				t.Errorf("parseEnvLine(%q) = (%q, %q), want (%q, %q)", tc.raw, key, value, tc.key, tc.value)
			}
		})
	}
}
