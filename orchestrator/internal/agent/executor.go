// Copyright (c) 2026 Intern Village. All rights reserved.
// SPDX-License-Identifier: Proprietary

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ExecutionResult contains the result of executing a Claude CLI command.
type ExecutionResult struct {
	ExitCode   int
	LogPath    string
	TokenUsage int
	Duration   time.Duration
	Error      error
}

// ClaudeRun represents a running Claude CLI process.
// Use Wait() to block until completion and get the result.
type ClaudeRun struct {
	LogPath    string
	resultChan chan *ExecutionResult
	result     *ExecutionResult
}

// Wait blocks until the Claude process completes and returns the result.
// Safe to call multiple times - returns cached result after first call.
func (r *ClaudeRun) Wait() *ExecutionResult {
	if r.result != nil {
		return r.result
	}
	r.result = <-r.resultChan
	return r.result
}

// Executor manages Claude CLI process execution.
type Executor struct {
	dataDir string
}

// NewExecutor creates a new Executor.
func NewExecutor(dataDir string) *Executor {
	return &Executor{
		dataDir: dataDir,
	}
}

// ExecuteClaudeAsync starts the Claude CLI and returns immediately with a ClaudeRun handle.
// The log file is created before returning, so log tailing can start immediately.
// Call Wait() on the returned ClaudeRun to block until completion.
func (e *Executor) ExecuteClaudeAsync(ctx context.Context, workDir, promptPath, projectID, taskID, subtaskID string, attemptNumber int) (*ClaudeRun, error) {
	startTime := time.Now()

	// Create log directory
	logDir := e.getLogDir(projectID, taskID, subtaskID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file
	logPath := filepath.Join(logDir, fmt.Sprintf("run-%03d.log", attemptNumber))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header to log file
	header := fmt.Sprintf("=== Agent Run %d ===\nStarted: %s\nWorking Directory: %s\nPrompt: %s\n\n",
		attemptNumber, startTime.Format(time.RFC3339), workDir, promptPath)
	if _, err := logFile.WriteString(header); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to write log header: %w", err)
	}

	// Flush to ensure header is visible to tailer
	if err := logFile.Sync(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to sync log file: %w", err)
	}

	// Read prompt content
	promptContent, err := os.ReadFile(promptPath) //nolint:gosec // promptPath is constructed by our code
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Build the command: echo the prompt and pipe to claude
	// Using stream-json output format for real-time log streaming
	// The --verbose flag is required for stream-json mode
	cmd := exec.CommandContext(ctx, "claude", "--print", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose") //nolint:gosec // Command is fixed
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(string(promptContent))

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Create result channel
	resultChan := make(chan *ExecutionResult, 1)

	// Run the rest in a goroutine
	go func() {
		defer logFile.Close()

		// Capture output with timestamps
		var wg sync.WaitGroup
		var outputBuffer strings.Builder
		var mu sync.Mutex // Protect concurrent writes to logFile

		// captureStreamJSON parses the stream-json output format from Claude CLI
		// and extracts meaningful content for logging
		captureStreamJSON := func(reader io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(reader)
			// Increase buffer size for long lines (JSON events can be large)
			buf := make([]byte, 0, 256*1024)
			scanner.Buffer(buf, 4*1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				timestamp := time.Now().Format("15:04:05")
				logLine := parseStreamJSONLine(line, timestamp)
				if logLine != "" {
					mu.Lock()
					//nolint:errcheck // Best effort logging
					logFile.WriteString(logLine)
					logFile.Sync() // Flush to disk for real-time visibility
					mu.Unlock()
					outputBuffer.WriteString(line + "\n")
				}
			}
		}

		// captureStderr handles stderr output (errors, warnings)
		captureStderr := func(reader io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(reader)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				timestamp := time.Now().Format("15:04:05")
				logLine := fmt.Sprintf("[%s] [STDERR] %s\n", timestamp, line)
				mu.Lock()
				//nolint:errcheck // Best effort logging
				logFile.WriteString(logLine)
				logFile.Sync()
				mu.Unlock()
			}
		}

		wg.Add(2)
		go captureStreamJSON(stdout)
		go captureStderr(stderr)

		// Wait for output capture to complete
		wg.Wait()

		// Wait for command to finish
		cmdErr := cmd.Wait()
		duration := time.Since(startTime)

		// Determine exit code
		exitCode := 0
		if cmdErr != nil {
			if exitErr, ok := cmdErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		// Write footer to log file
		footer := fmt.Sprintf("\n=== Run Complete ===\nDuration: %s\nExit Code: %d\n",
			duration.String(), exitCode)
		//nolint:errcheck // Best effort logging
		logFile.WriteString(footer)

		// Parse token usage from output
		tokenUsage := parseTokenUsage(outputBuffer.String())

		resultChan <- &ExecutionResult{
			ExitCode:   exitCode,
			LogPath:    logPath,
			TokenUsage: tokenUsage,
			Duration:   duration,
			Error:      cmdErr,
		}
	}()

	return &ClaudeRun{
		LogPath:    logPath,
		resultChan: resultChan,
	}, nil
}

// ExecuteClaude runs the Claude CLI with the given prompt file.
// It captures stdout/stderr to a log file and returns the execution result.
// This is a blocking call - use ExecuteClaudeAsync for non-blocking execution.
func (e *Executor) ExecuteClaude(ctx context.Context, workDir, promptPath, projectID, taskID, subtaskID string, attemptNumber int) (*ExecutionResult, error) {
	run, err := e.ExecuteClaudeAsync(ctx, workDir, promptPath, projectID, taskID, subtaskID, attemptNumber)
	if err != nil {
		return nil, err
	}
	return run.Wait(), nil
}

// getLogDir returns the log directory path for a subtask.
func (e *Executor) getLogDir(projectID, taskID, subtaskID string) string {
	if subtaskID != "" {
		return filepath.Join(e.dataDir, "logs", projectID, taskID, subtaskID)
	}
	return filepath.Join(e.dataDir, "logs", projectID, taskID)
}

// GetLogPath returns the expected log file path for a run.
func (e *Executor) GetLogPath(projectID, taskID, subtaskID string, attemptNumber int) string {
	logDir := e.getLogDir(projectID, taskID, subtaskID)
	return filepath.Join(logDir, fmt.Sprintf("run-%03d.log", attemptNumber))
}

// ReadLogFile reads the content of a log file.
func (e *Executor) ReadLogFile(logPath string) (string, error) {
	content, err := os.ReadFile(logPath) //nolint:gosec // logPath is validated
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}
	return string(content), nil
}

// streamEvent represents a JSON event from Claude CLI stream-json output
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
	Result  string `json:"result,omitempty"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
			Name string `json:"name,omitempty"`
		} `json:"content,omitempty"`
	} `json:"message,omitempty"`
	ToolName string         `json:"tool_name,omitempty"`
	Input    map[string]any `json:"input,omitempty"`
	Usage    struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

// parseStreamJSONLine parses a single line of stream-json output and returns
// a human-readable log line, or empty string if the event should be skipped.
func parseStreamJSONLine(line, timestamp string) string {
	var event streamEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		// Not valid JSON, log as-is
		return fmt.Sprintf("[%s] %s\n", timestamp, line)
	}

	switch event.Type {
	case "system":
		switch event.Subtype {
		case "init":
			return fmt.Sprintf("[%s] ⚡ Claude session initialized\n", timestamp)
		case "hook_response":
			// Skip hook responses (noisy)
			return ""
		default:
			return ""
		}

	case "assistant":
		// Extract text content from assistant messages
		var texts []string
		for _, content := range event.Message.Content {
			if content.Type == "text" && content.Text != "" {
				texts = append(texts, content.Text)
			}
		}
		if len(texts) > 0 {
			return fmt.Sprintf("[%s] 💬 %s\n", timestamp, strings.Join(texts, " "))
		}
		return ""

	case "user":
		// User messages are typically tool results, can be verbose
		return ""

	case "tool_use":
		// Log tool usage
		if event.ToolName != "" {
			switch event.ToolName {
			case "Read":
				if path, ok := event.Input["file_path"].(string); ok {
					return fmt.Sprintf("[%s] 📖 Reading: %s\n", timestamp, path)
				}
			case "Edit":
				if path, ok := event.Input["file_path"].(string); ok {
					return fmt.Sprintf("[%s] ✏️  Editing: %s\n", timestamp, path)
				}
			case "Write":
				if path, ok := event.Input["file_path"].(string); ok {
					return fmt.Sprintf("[%s] 📝 Writing: %s\n", timestamp, path)
				}
			case "Bash":
				if cmd, ok := event.Input["command"].(string); ok {
					// Truncate long commands
					if len(cmd) > 100 {
						cmd = cmd[:100] + "..."
					}
					return fmt.Sprintf("[%s] 💻 Running: %s\n", timestamp, cmd)
				}
			case "Glob":
				if pattern, ok := event.Input["pattern"].(string); ok {
					return fmt.Sprintf("[%s] 🔍 Searching: %s\n", timestamp, pattern)
				}
			case "Grep":
				if pattern, ok := event.Input["pattern"].(string); ok {
					return fmt.Sprintf("[%s] 🔎 Grepping: %s\n", timestamp, pattern)
				}
			case "Task":
				if desc, ok := event.Input["description"].(string); ok {
					return fmt.Sprintf("[%s] 🤖 Spawning agent: %s\n", timestamp, desc)
				}
			default:
				return fmt.Sprintf("[%s] 🔧 Using tool: %s\n", timestamp, event.ToolName)
			}
		}
		return ""

	case "result":
		// Final result with token usage
		if event.Subtype == "success" {
			return fmt.Sprintf("[%s] ✅ Task completed successfully\n", timestamp)
		} else if event.Subtype == "error" {
			return fmt.Sprintf("[%s] ❌ Task failed: %s\n", timestamp, event.Result)
		}
		return ""

	default:
		// Skip unknown event types
		return ""
	}
}

// parseTokenUsage attempts to parse token usage from Claude CLI stream-json output.
// It looks for the "result" event which contains usage information.
func parseTokenUsage(output string) int {
	// First try to parse from stream-json result events
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event struct {
			Type  string `json:"type"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			if event.Type == "result" && (event.Usage.InputTokens > 0 || event.Usage.OutputTokens > 0) {
				return event.Usage.InputTokens + event.Usage.OutputTokens
			}
		}
	}

	// Fall back to legacy pattern matching for non-JSON output
	patterns := []string{
		`(?i)total\s+tokens[:\s]+(\d+)`,
		`(?i)tokens\s+used[:\s]+(\d+)`,
		`(?i)tokens[:\s]+(\d+)`,
		`(?i)input[:\s]+(\d+)\s+tokens`,
		`(?i)output[:\s]+(\d+)\s+tokens`,
	}

	totalTokens := 0
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(output, -1)
		for _, match := range matches {
			if len(match) > 1 {
				if count, err := strconv.Atoi(match[1]); err == nil {
					totalTokens += count
				}
			}
		}
	}

	return totalTokens
}
