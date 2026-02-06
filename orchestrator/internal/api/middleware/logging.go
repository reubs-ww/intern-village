// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
// It also implements http.Flusher to support SSE streaming.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Flush implements http.Flusher, required for SSE streaming.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logger is a middleware that logs HTTP requests using zerolog.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := wrapResponseWriter(w)

		// Get request ID from chi middleware
		reqID := chimw.GetReqID(r.Context())

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log the request
		duration := time.Since(start)

		logEvent := log.Info()
		if wrapped.status >= 500 {
			logEvent = log.Error()
		} else if wrapped.status >= 400 {
			logEvent = log.Warn()
		}

		logEvent.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Int("status", wrapped.status).
			Dur("duration", duration).
			Str("request_id", reqID).
			Msg("HTTP request")
	})
}
