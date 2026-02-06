#!/bin/bash
# Claude Code wrapper with exit signal support

SIGNAL_FILE="/tmp/claude-exit-signal"

# Clean up any old signal file
rm -f "$SIGNAL_FILE"

# Cleanup on script exit
cleanup() {
  rm -f "$SIGNAL_FILE"
}
trap cleanup EXIT

# Read stdin if available (for piping)
if [ ! -t 0 ]; then
  PROMPT=$(cat)
fi

# Start Claude Code in foreground, but monitor in background
if [ -n "$PROMPT" ]; then
  echo "$PROMPT" | claude --dangerously-skip-permissions "$@" &
else
  claude --dangerously-skip-permissions "$@" &
fi
CLAUDE_PID=$!

# Monitor for exit signal
while kill -0 $CLAUDE_PID 2>/dev/null; do
  if [ -f "$SIGNAL_FILE" ]; then
    echo ""
    echo "Exit signal received. Goodbye!"
    rm -f "$SIGNAL_FILE"
    kill $CLAUDE_PID 2>/dev/null
    wait $CLAUDE_PID 2>/dev/null
    exit 0
  fi
  sleep 0.3
done

# Wait for Claude to finish naturally
wait $CLAUDE_PID
