// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

//go:build embed_frontend

package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed frontend/dist
var frontendFS embed.FS

// FrontendHandler serves the embedded frontend SPA.
type FrontendHandler struct {
	fileServer http.Handler
	indexHTML  []byte
}

// NewFrontendHandler creates a new handler for serving the frontend.
func NewFrontendHandler() (*FrontendHandler, error) {
	// Get the dist subdirectory
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		return nil, err
	}

	// Read index.html for SPA fallback
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		return nil, err
	}

	return &FrontendHandler{
		fileServer: http.FileServer(http.FS(distFS)),
		indexHTML:  indexHTML,
	}, nil
}

// ServeHTTP handles frontend requests.
// For static assets, it serves them directly.
// For other routes, it returns index.html for SPA routing.
func (h *FrontendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Skip API routes
	if strings.HasPrefix(path, "/api") || path == "/health" {
		http.NotFound(w, r)
		return
	}

	// Check if the file exists in the embedded FS
	distFS, _ := fs.Sub(frontendFS, "frontend/dist")
	cleanPath := strings.TrimPrefix(path, "/")
	if cleanPath == "" {
		cleanPath = "index.html"
	}

	// Try to open the file
	f, err := distFS.Open(cleanPath)
	if err == nil {
		f.Close()
		// File exists, serve it
		h.fileServer.ServeHTTP(w, r)
		return
	}

	// File doesn't exist, serve index.html for SPA routing
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(h.indexHTML)
}
