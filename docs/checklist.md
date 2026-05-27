# Checklist

The checklist of things to implement.

## Scaffolding

- [x] Create root `main.go` (thin entry point: build app, run)
- [x] Set up `internal/ui` package with minimal Model, Init, Update, View
- [x] `tea.NewProgram(...).Run()` wired up from `main.go`
- [x] First table-driven unit tests for the model

## Storage and indexing

- [ ] `internal/store`: open DB, schema (`sessions`, `memories`, FTS tables, `_ai`/`_ad`/`_au` triggers)
- [ ] `internal/indexer`: orchestrator and per-file sync decisions (mtime + size)
- [ ] Session JSONL parser: title, project, date, size, msg count, tokens, duration
- [ ] Session incremental parse via stored byte offset
- [ ] Memory parser: YAML frontmatter (yaml.v3) plus body
- [ ] `MEMORY.md` line operations (read order, rename, delete) with mtime + retry-once
- [ ] Initial corpus build on startup with a spinner

## File watching

- [ ] `internal/watcher`: fsnotify over `~/.claude/projects/`
- [ ] 500ms per-file debounce
- [ ] Watch each project's `memory/` subdir
- [ ] Watch `MEMORY.md` for ordering refresh (not indexed)
- [ ] `EntityChange` channel and `tea.Cmd` that re-registers after each message
- [ ] UI reacts to `EntityChange`: refresh list, refresh preview if the affected row is selected

## Sessions tab

- [ ] List view with weighted columns (title:5, project:5, date:1, size:1, msgs:1)
- [ ] Column sizing reactive to terminal width and layout
- [ ] Cursor and scroll
- [ ] Sort cycling (`ctrl-s`)
- [ ] Layout cycling (`ctrl-/`): up -> right -> down -> left -> hidden
- [ ] Preview pane: metadata block (model, duration, tokens) and colored conversation transcript
- [ ] Preview scroll (`ctrl-u/d`)
- [ ] Resume via `claude --resume <uuid>` on `enter` (`tea.ExecProcess`)
- [ ] Pager mode (`ctrl-o`, `less`-style full preview)
- [ ] Rename: inline text input at row position (`ctrl-r`)
- [ ] Two-step delete: `ctrl-x`, tab to select, enter to confirm

## Memories tab

- [ ] List view with columns: NAME, PROJECT, TYPE, DATE, SIZE
- [ ] Sort cycling (per-tab fields)
- [ ] Glamour-rendered Markdown preview with raw frontmatter at top
- [ ] Type color accent (cyan, yellow, green, blue)
- [ ] Open in `$EDITOR` via `tea.ExecProcess` (`enter`)
- [ ] Rename (`ctrl-r`) and rewrite `MEMORY.md`
- [ ] Two-step delete (`ctrl-x`) and rewrite `MEMORY.md`

## Search

- [ ] Unified `/` search with scope cycling (`tab` or `ctrl-t`)
- [ ] Scope label in prompt (`Search [sessions]:` etc.)
- [ ] Per-corpus FTS5 query for single-tab scope
- [ ] `UNION ALL` cross-corpus query for global scope
- [ ] Lipgloss snippet styling (split on `char(2)` and `char(3)` sentinels)
- [ ] Switch tab on selecting a global result from the inactive tab
- [ ] Viewport's built-in `/` inside the preview pane

## Cross-cutting

- [ ] Add `internal/logging` (slog file handler at `$XDG_STATE_HOME/claudex/`)
- [ ] `WindowSizeMsg` cascade to all subcomponents
- [ ] Help screen (`?`) via bubbles `help` and `key.Binding`
- [ ] Sealed-interface state types for screen lifecycles (loading, loaded, failed)
- [ ] Graceful quit (close watcher, close DB)
- [ ] Session move between projects (`mv` and rewrite `.cwd` on every line)
- [ ] Memory move between projects (`mv` and update both source and destination `MEMORY.md`)

## Revisit in v2

- [ ] Mouse support
- [ ] Persistent preferences (active tab, sort order, last selection per tab, layout)
- [ ] Subagent conversation viewing (`<uuid>/subagents/`)
