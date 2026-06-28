package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"

	"backend/auth"
)

func newNavidromeProxy(socket string, logger *zap.Logger) *httputil.ReverseProxy {
	target, _ := url.Parse("http://navidrome")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}
	proxy.FlushInterval = -1

	orig := proxy.Director
	proxy.Director = func(r *http.Request) {
		orig(r)
		r.Header.Del(userHeader)
		if user, ok := r.Context().Value(userKey).(string); ok && user != "" {
			r.Header.Set(userHeader, user)
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		logger.Error("proxy to navidrome failed", zap.Error(err))
		http.Error(w, "navidrome unavailable", http.StatusBadGateway)
	}
	return proxy
}

func webHandler(oidc *auth.OIDC, proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := oidc.SessionUser(r)
		if !ok {
			if isBrowserNavigation(r) {
				http.Redirect(w, r, "/syncloud-oidc/login", http.StatusFound)
				return
			}
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		proxy.ServeHTTP(w, withUser(r, user))
	})
}

func isBrowserNavigation(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	return wantsHTML(r.Header.Get("Accept"))
}

func wantsHTML(accept string) bool {
	for _, part := range splitComma(accept) {
		if part == "text/html" || part == "application/xhtml+xml" {
			return true
		}
	}
	return false
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' || s[i] == ';' {
			out = append(out, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, trimSpace(s[start:]))
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
