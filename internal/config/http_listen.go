package config

import (
	"fmt"
	"net"
	"strconv"
)

func resolveHTTPListenAddr(src envSource) (host string, port int, addr string, err error) {
	_, hostSet := src.resolved("HTTP_HOST")
	_, portSet := src.resolved("HTTP_PORT")

	if !hostSet && !portSet {
		if legacy, ok := src.resolved("HTTP_ADDR"); ok && legacy != "" {
			h, p, err := net.SplitHostPort(legacy)
			if err != nil {
				return "", 0, "", fmt.Errorf("HTTP_ADDR: %w", err)
			}
			portNum, err := strconv.Atoi(p)
			if err != nil || portNum < 1 || portNum > 65535 {
				return "", 0, "", fmt.Errorf("HTTP_ADDR: invalid port %q", p)
			}
			return h, portNum, legacy, nil
		}
	}

	host = src.envOr("HTTP_HOST", "127.0.0.1")
	if host == "" {
		return "", 0, "", fmt.Errorf("HTTP_HOST must not be empty")
	}

	port, err = src.envInt("HTTP_PORT", 8081)
	if err != nil {
		return "", 0, "", err
	}
	if port < 1 || port > 65535 {
		return "", 0, "", fmt.Errorf("HTTP_PORT must be 1-65535, got %d", port)
	}

	addr = net.JoinHostPort(host, strconv.Itoa(port))
	return host, port, addr, nil
}

// DocsBaseURL is a host clients on this machine can open in a browser (not 0.0.0.0).
func DocsBaseURL(host string, port int) string {
	h := host
	switch host {
	case "0.0.0.0", "::", "[::]":
		h = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(h, strconv.Itoa(port))
}
