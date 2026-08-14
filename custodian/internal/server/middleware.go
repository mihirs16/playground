package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	publicPrefix = "/v1/"
	adminPrefix  = "/admin/"
)

// recoverMiddleware turns a panic in any handler into a 500 problem document
// rather than a dropped connection, and logs it.
func recoverMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("handler panic", "path", r.URL.Path, "panic", recovered)
					writeProblem(w, http.StatusInternalServerError, "internal", "internal error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// accessLogMiddleware logs every request. Admin-surface access is logged at a
// level that makes an unrecognised /admin/* call a visible leak signal.
func accessLogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"client_ip", clientIP(r),
				"duration_ms", time.Since(start).Milliseconds(),
			}
			if strings.HasPrefix(r.URL.Path, adminPrefix) {
				logger.Info("admin access", attrs...)
			} else {
				logger.Debug("request", attrs...)
			}
		})
	}
}

// corsMiddleware serves the public surface's explicit-allowlist CORS. It never
// answers a wildcard, exposes ETag so browsers can make conditional requests,
// and short-circuits preflight. It touches only /v1/* — the admin surface has
// no CORS at all.
func corsMiddleware(allowlist []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowlist))
	for _, origin := range allowlist {
		allowed[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, publicPrefix) {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "If-None-Match")
				w.Header().Set("Access-Control-Expose-Headers", "ETag")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// adminAuthMiddleware guards only /admin/*. It accepts a single long-lived
// bearer token, compares its SHA-256 against the configured hash in constant
// time, and holds no plaintext token anywhere.
func adminAuthMiddleware(tokenHash string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, adminPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			if !validBearer(r.Header.Get("Authorization"), tokenHash) {
				writeProblem(w, http.StatusUnauthorized, "unauthorized", "a valid admin bearer token is required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validBearer(header, wantHash string) bool {
	if wantHash == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	gotHash := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(gotHash), []byte(wantHash)) == 1
}

// clientIP reports the real client address, preferring the leftmost entry of
// CloudFront's X-Forwarded-For chain over the direct peer.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if comma := strings.IndexByte(xff, ','); comma >= 0 {
			return strings.TrimSpace(xff[:comma])
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}
