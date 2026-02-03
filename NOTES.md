# Claude JSONL Reader - Developer Notes

## Project Status: Working MVP

The viewer is functional with two modes. All tests pass. Ready for real-world testing and iteration.

## Quick Start

```bash
go build -o claude-jsonl-reader
./claude-jsonl-reader              # Run in any project directory
cd examples && ../claude-jsonl-reader  # Test with example files
```

## Architecture

```
main.go         Entry point, resolves Claude history directory
model.go        Bubble Tea model, handles all UI state and rendering
message.go      Message parsing and type-specific rendering
jsonl.go        Raw JSON parsing with nested JSON expansion
files.go        File discovery, Claude project path resolution
highlight.go    JSON syntax highlighting, search highlighting
preview.go      Right-pane preview (JSON mode only)
```

## Two View Modes

### JSON Mode (original)
- Two-column layout: JSON on left, string preview on right
- Line-by-line navigation
- Syntax highlighting, search highlighting
- Good for debugging raw data

### Message Mode (new)
- Single-column, full-width
- Message-by-message navigation
- Type-aware rendering (user/assistant/system/summary)
- Starts in this mode by default

Toggle with `Tab`.

## JSONL Schema (from Claude Code)

Key message types found in examples:
- `user` - User messages, can have `tool_result` content
- `assistant` - Assistant responses with `text`, `thinking`, `tool_use` blocks
- `system` - System messages (subtype: local_command, etc.)
- `summary` - Session summaries
- `file-history-snapshot` - Skipped in message mode

Content can be:
- A string (direct markdown/text)
- An array of content blocks with `type` field

## Known Issues / TODO

1. **Scroll within long messages**: Currently `j/k` moves between messages. Need a way to scroll within a long message (maybe `Ctrl-d/u` for intra-message scroll?)

2. **Search in message mode**: Search (`/`) finds text but doesn't highlight in message mode. Need to implement search highlighting for rendered messages.

3. **Meta messages**: Messages with `isMeta: true` (like skill loading) are rendered but could be collapsed or hidden by default.

4. **Thinking blocks**: Currently shown in full. Could add a toggle to show/hide thinking blocks (they can be very long).

5. **Tool use rendering**: Shows prettified JSON input, but could be smarter about common tools (e.g., show file paths for Read tool, show command for Bash tool).

6. **Performance**: Large JSONL files parse twice (once for JSON mode, once for message mode). Could lazy-load or unify parsing.

7. **Assistant streaming**: Some assistant messages have `stop_reason: null` indicating incomplete streaming. Not currently handled specially.

## File Structure for Claude History

Claude stores history in `~/.claude/projects/<cleaned-path>/`

Path cleaning: `/Users/foo/my-project` → `-Users-foo-my-project`

The viewer auto-detects this when run from a project directory.

## Testing

```bash
go test ./...           # Run all tests
go test -v ./...        # Verbose output
```

Example files are in `examples/` - real Claude Code history samples.

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/glamour` - Markdown rendering
- `github.com/muesli/reflow/wordwrap` - Text wrapping

## Code Patterns

- Model updates return `(tea.Model, tea.Cmd)` - Bubble Tea pattern
- Rendering functions build strings, don't modify state
- Message parsing is separate from rendering for testability
- Type switches on `msg.Type` for different rendering paths

## Questions for Product

1. Should `file-history-snapshot` entries be visible at all?
2. Should meta messages be hidden by default?
3. What's the ideal UX for very long messages (multi-page)?
4. Should there be a "collapsed" view showing just message types/timestamps?
