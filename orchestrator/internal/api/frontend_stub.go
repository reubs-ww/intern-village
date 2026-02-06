// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

//go:build !embed_frontend

package api

import (
	"errors"
	"net/http"
)

// FrontendHandler is a stub when frontend is not embedded.
type FrontendHandler struct{}

// NewFrontendHandler returns an error when frontend is not embedded.
func NewFrontendHandler() (*FrontendHandler, error) {
	return nil, errors.New("frontend not embedded (build without embed_frontend tag)")
}

// ServeHTTP is a no-op stub.
func (h *FrontendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
