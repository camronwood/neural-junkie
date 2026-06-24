package main

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

// resolveListenAddr picks the HTTP bind address. Default is loopback-only (127.0.0.1:18765).
// Set NEURAL_JUNKIE_LISTEN_ALL=1 or SERVER_HOST=0.0.0.0 to listen on all interfaces.
// The -addr flag overrides when not the compiled default ":18765".
func resolveListenAddr(flagAddr string, cfg *config.Config) string {
	if flagAddr != ":18765" {
		return flagAddr
	}
	host := "127.0.0.1"
	if cfg != nil && strings.TrimSpace(cfg.Server.Host) != "" {
		host = strings.TrimSpace(cfg.Server.Host)
		if strings.EqualFold(host, "localhost") {
			host = "127.0.0.1"
		}
	}
	if os.Getenv("NEURAL_JUNKIE_LISTEN_ALL") == "1" {
		host = "0.0.0.0"
	}
	port := 18765
	if cfg != nil && cfg.Server.Port != 0 {
		port = cfg.Server.Port
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// hubPublicHost returns host:port for log lines and agent URLs (localhost when bound to loopback).
func hubPublicHost(listenAddr string) string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return listenAddr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	if host == "127.0.0.1" || host == "::1" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}

func corsAllowsOrigin(origin string) bool {
	if os.Getenv("NEURAL_JUNKIE_CORS_ANY") == "1" {
		return true
	}
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if strings.HasSuffix(host, ".localhost") {
		return true
	}
	// Tauri / Vite dev
	if host == "localhost" && (u.Port() == "1420" || u.Port() == "5173") {
		return true
	}
	for _, extra := range strings.Split(os.Getenv("NEURAL_JUNKIE_CORS_ORIGINS"), ",") {
		if strings.TrimSpace(extra) == origin {
			return true
		}
	}
	return false
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if corsAllowsOrigin(origin) {
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-NJ-Hub-Token, X-NJ-Session, X-NJ-Bootstrap")
	}
}

// localOnly rejects non-loopback clients unless NEURAL_JUNKIE_HUB_TOKEN is configured and sent.
func localOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !hub.RequireHubAccess(w, r) {
			return
		}
		next(w, r)
	}
}
