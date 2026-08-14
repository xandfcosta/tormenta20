package api

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// loadEnvFile exports every KEY=VALUE of path that is NOT already set in the
// process environment, and returns without error when the file does not exist.
//
// The process wins on purpose (ALE-119): the file carries the environment's
// defaults, so a one-off `PORT=4000 pnpm start` — or a CI runner that exports
// everything and ships no file at all — still overrides it without editing
// anything on disk.
//
//	loadEnvFile(".env.production") // → PORT, JWT_SECRET, STATIC_DIR… exported
func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("env file %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		key, value, err := parseEnvLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if key == "" || os.Getenv(key) != "" {
			continue // blank/comment line, or already exported: the process wins
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s:%d: set %s: %w", path, line, key, err)
		}
	}
	return scanner.Err()
}

// parseEnvLine splits one KEY=VALUE line. A blank or `#` line yields an empty
// key and no error; anything else without a key or an `=` is a typo worth
// failing the boot for, because a silently skipped line means a default the
// operator believes they changed.
func parseEnvLine(raw string) (string, string, error) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", nil
	}
	key, value, found := strings.Cut(line, "=")
	key = strings.TrimSpace(key)
	if !found || key == "" {
		return "", "", fmt.Errorf("expected KEY=VALUE, got %q", raw)
	}
	return key, unquote(strings.TrimSpace(value)), nil
}

// unquote strips one layer of matching surrounding quotes, so a value can keep
// leading/trailing spaces or a trailing `#` that would otherwise read as noise.
func unquote(value string) string {
	for _, quote := range []string{`"`, `'`} {
		if len(value) >= 2 && strings.HasPrefix(value, quote) && strings.HasSuffix(value, quote) {
			return value[1 : len(value)-1]
		}
	}
	return value
}
