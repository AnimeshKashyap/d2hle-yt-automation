package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// envSource resolves settings with precedence: .env (highest), then OS environment, then Load() defaults.
type envSource struct {
	file map[string]string
}

func loadEnvSource() envSource {
	path := findDotEnvPath()
	if path == "" {
		return envSource{}
	}
	m, err := parseDotEnvFile(path)
	if err != nil || m == nil {
		return envSource{}
	}
	return envSource{file: m}
}

// findDotEnvPath returns the first existing .env: current working directory, then beside the executable.
func findDotEnvPath() string {
	candidates := []string{".env"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func parseDotEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	out := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := parseDotEnvLine(line)
		if !ok {
			continue
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseDotEnvLine(line string) (key, val string, ok bool) {
	i := strings.Index(line, "=")
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	if key == "" {
		return "", "", false
	}
	val = strings.TrimSpace(line[i+1:])
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}
	return key, val, true
}

func (src envSource) resolved(key string) (string, bool) {
	if src.file != nil {
		if v, ok := src.file[key]; ok {
			return v, true
		}
	}
	if v := os.Getenv(key); v != "" {
		return v, true
	}
	return "", false
}

func (src envSource) envOr(key, def string) string {
	v, ok := src.resolved(key)
	if !ok {
		return def
	}
	if v == "" {
		return def
	}
	return v
}

func (src envSource) envString(key string) string {
	v, ok := src.resolved(key)
	if ok {
		return v
	}
	return ""
}

func (src envSource) envInt(key string, def int) (int, error) {
	v, ok := src.resolved(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

func (src envSource) envBool(key string, def bool) (bool, error) {
	v, ok := src.resolved(key)
	if !ok || v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func (src envSource) envDuration(key string, def time.Duration) (time.Duration, error) {
	v, ok := src.resolved(key)
	if !ok || v == "" {
		return def, nil
	}
	return parseDurationValue(key, v)
}

func parseDurationValue(key, v string) (time.Duration, error) {
	if strings.HasSuffix(v, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "m"))
		if err != nil {
			return 0, fmt.Errorf("%s: %w", key, err)
		}
		return time.Duration(n) * time.Minute, nil
	}
	if strings.HasSuffix(v, "s") {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "s"))
		if err != nil {
			return 0, fmt.Errorf("%s: %w", key, err)
		}
		return time.Duration(n) * time.Second, nil
	}
	sec, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return time.Duration(sec) * time.Second, nil
}
