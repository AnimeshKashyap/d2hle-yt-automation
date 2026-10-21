package config

import (
	"testing"
)

func TestResolveHTTPListenHostPort(t *testing.T) {
	src := envSource{file: map[string]string{
		"HTTP_HOST": "0.0.0.0",
		"HTTP_PORT": "9090",
	}}
	host, port, addr, err := resolveHTTPListenAddr(src)
	if err != nil {
		t.Fatal(err)
	}
	if host != "0.0.0.0" || port != 9090 || addr != "0.0.0.0:9090" {
		t.Fatalf("got host=%q port=%d addr=%q", host, port, addr)
	}
}

func TestResolveHTTPListenLegacyAddr(t *testing.T) {
	src := envSource{file: map[string]string{
		"HTTP_ADDR": "127.0.0.1:7777",
	}}
	_, port, addr, err := resolveHTTPListenAddr(src)
	if err != nil {
		t.Fatal(err)
	}
	if port != 7777 || addr != "127.0.0.1:7777" {
		t.Fatalf("got port=%d addr=%q", port, addr)
	}
}

func TestDocsBaseURLWildcardHost(t *testing.T) {
	if u := DocsBaseURL("0.0.0.0", 8081); u != "http://127.0.0.1:8081" {
		t.Fatalf("got %q", u)
	}
}
