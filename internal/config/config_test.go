package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func chdirTo(t *testing.T, dir string) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func isolatedDir(t *testing.T) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(thisFile), "testdata", "scratch")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "run-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestLoadDefaults(t *testing.T) {
	dir := isolatedDir(t)
	chdirTo(t, dir)
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConcurrentJobs != 1 {
		t.Fatalf("expected max jobs 1, got %d", cfg.MaxConcurrentJobs)
	}
	if len(cfg.AllowedMediaRoots) != 1 {
		t.Fatalf("expected one media root, got %v", cfg.AllowedMediaRoots)
	}
}

func TestDotEnvOverridesEnvironment(t *testing.T) {
	dir := isolatedDir(t)
	chdirTo(t, dir)
	os.Clearenv()
	t.Setenv("DATA_ROOT", "/from-env")
	t.Setenv("HTTP_HOST", "0.0.0.0")
	t.Setenv("HTTP_PORT", "9999")

	if err := os.WriteFile(".env", []byte("DATA_ROOT=./from-dotenv\nHTTP_HOST=127.0.0.1\nHTTP_PORT=7777\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataRoot != "./from-dotenv" {
		t.Fatalf("DATA_ROOT: got %q", cfg.DataRoot)
	}
	if cfg.HTTPAddr != "127.0.0.1:7777" {
		t.Fatalf("HTTP listen: got %q", cfg.HTTPAddr)
	}
}

func TestParseDotEnvLine(t *testing.T) {
	k, v, ok := parseDotEnvLine(`FOO="bar baz"`)
	if !ok || k != "FOO" || v != "bar baz" {
		t.Fatalf("quoted: %q %q %v", k, v, ok)
	}
	_, _, ok = parseDotEnvLine("# comment")
	if ok {
		t.Fatal("comment should not parse")
	}
}

func TestFindDotEnvPrefersWorkingDirectory(t *testing.T) {
	dir := isolatedDir(t)
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DATA_ROOT=./parent\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	chdirTo(t, sub)
	os.Clearenv()
	if err := os.WriteFile(".env", []byte("DATA_ROOT=./cwd-root\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataRoot != "./cwd-root" {
		t.Fatalf("expected cwd .env, got %q", cfg.DataRoot)
	}
}
