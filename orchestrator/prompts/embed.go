// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

// Package prompts provides embedded prompt templates for agents.
package prompts

import "embed"

// FS contains the embedded prompt template files.
//
//go:embed *.md
var FS embed.FS
