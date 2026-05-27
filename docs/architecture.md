# Architecture

Claudex is a Bubbletea TUI for browsing Claude Code session history and memory files. This document contains the design and the decisions behind it.

Background research is in `docs/workspace`. Where the research and this document disagree, follow this document.

## Goals (v1)

1. Full feature parity with proof-of-concept session explorer (Bash).
2. Equivalent feature set for memories: browse, preview, rename, delete, search, and external edit.
3. Full-text search across entire session conversations and memory bodies, not just the first 500 chars as in the bash version.
4. Live updates as sessions and memories change on disk. Required, because the user runs two or three concurrent Claude sessions throughout the workday.
5. Zero runtime dependencies (no fzf, jq, fd, rg, less).
6. TDD via `charmbracelet/x/exp/teatest`. Each feature has a corresponding test.

## Package layout

```
claudex/
├── main.go                 Constructs the model and runs the program.
├── internal/
│   ├── ui/                 Renders the UI and handles input.
│   ├── store/              Opens SQLite and runs FTS5 queries.
│   ├── indexer/            Syncs file changes into the database.
│   ├── watcher/            Watches files with fsnotify and debounces bursts.
│   ├── format/             Sizes columns and formats values for display.
│   └── logging/            Sets up the slog logger.
├── docs/
│   ├── architecture.md     Describes the design (this file).
│   ├── checklist.md        Tracks the features to build.
│   └── workspace/          Holds local-only notes (gitignored).
└── go.mod
```

Dependency direction, compiler-enforced via `internal/`:

- `main` imports `ui` and `logging`.
- `ui` imports `store`, `format`, and `watcher` (it consumes `EntityChange` messages emitted by the watcher).
- `indexer` imports `store`.
- `watcher` imports nothing inside `internal/`; it emits events over a channel.
- `logging` imports nothing inside `internal/`. Only `main` imports `logging`; every other package uses `log/slog`'s default logger directly.

Packages are introduced organically as features arrive. The initial scaffold is just `main.go` and a minimal `internal/ui/`. The layout above is where the code is headed, not day-one boilerplate.

## Storage and search

The driver is `modernc.org/sqlite`, which is pure Go (no CGO) and includes FTS5. The database is at `~/.cache/claudex/index.db`. It's a derived cache and safe to delete; we rebuild it on the next startup.

The schema has two parent tables (`sessions`, `memories`) and two FTS5 tables (`sessions_fts`, `memories_fts`) in external-content mode (`content='<parent>'`). Triggers named `_ai`, `_ad`, and `_au` (after-insert, after-delete, after-update) keep the inverted indexes in sync with the parent tables inside the same transaction.

BM25 ranking is corpus-relative, so a shared FTS table would let memory vocabulary distort session scores. That's why there are two FTS tables instead of one. Cross-corpus search uses `UNION ALL`; cross-rank scores aren't perfectly calibrated but are acceptable for UI search.

The tokenizer is `porter unicode61`. It does case folding, diacritic stripping, and English stemming, so "searching", "searched", and "search" all match.

### Snippet styling

SQL emits abstract sentinels around hits, and Go restyles them via Lipgloss:

```sql
snippet(sessions_fts, 1, char(2), char(3), '...', 30)
```

`char(2)` and `char(3)` (ASCII STX and ETX) can't collide with text content. Go reads the snippet, splits on the sentinels, and wraps each hit segment with a Lipgloss style. The alternative would be baking ANSI codes into the SQL markers, but doing it in Go means theme changes don't require re-indexing, Lipgloss's width math stays correct (it strips ANSI for length), and snippets remain testable as plain strings.

## Sync strategy

For each file, we compare the stored `file_mtime` and `file_size` to the filesystem on each pass and act accordingly:

| Condition                | Action                                |
|--------------------------|---------------------------------------|
| File not in DB           | Full parse and insert                 |
| Sessions: size increased | Incremental parse from stored offset  |
| Memories: size increased | Full re-parse (files are small)       |
| Size decreased           | Full re-parse (sessions rewind, etc.) |
| mtime changed, same size | Full re-parse                         |
| Unchanged                | Skip                                  |
| In DB but file gone      | DELETE                                |

For sessions, the incremental parse stores the byte offset of the last parsed line and reads from there on size growth. The append-only assumption holds in normal cases; `custom-title` and `compact_boundary` messages are all appended. A rewind truncates the file, and the size-decreased row in the table covers that case with a full re-parse.

Format changes from Anthropic shouldn't break the byte-offset assumption, because each JSONL line is a self-contained JSON object. Format changes within a line don't invalidate offsets. Only a change that re-emits prior lines would, and that would be unusual.

## File watching

We watch the filesystem with `fsnotify` (inotify on Linux). Each project gets two watches: one on the project directory and one on its `memory/` subdirectory. With around ten active projects, that's roughly twenty watches, well below Linux's default inotify limit of 8192.

A 500ms per-file debounce collapses burst writes during live Claude responses, which can append dozens of lines per second.

`MEMORY.md` is watched too, but only so the memory list can refresh its ordering when the file changes. It is not indexed as a memory in the FTS tables.

The watcher emits `EntityChange{Kind, ID, Op}` over a channel, picked up by a `tea.Cmd` in the UI that re-registers itself after each message.

## UI

### Tabs

| Tab      | Default | Sort fields                      |
|----------|---------|----------------------------------|
| Sessions | yes     | TITLE, PROJECT, DATE, SIZE, MSGS |
| Memories |         | NAME, PROJECT, TYPE, DATE, SIZE  |

Each tab keeps its own cursor, sort order, selection set, and search state.

### Search

A single search bar covers both tabs, with a scope toggle.

- `/` opens search; the default scope is the current tab.
- Inside search, `tab` (or `ctrl-t`) cycles scope: current -> all -> other tab -> current.
- The prompt shows scope: `Search [sessions]:`, `Search [all]:`, or `Search [memories]:`.
- Selecting a global result from the inactive tab switches to that tab.
- Inside the preview pane, viewport's built-in `/` does an in-content search.

### Keybinds

| Key                 | Action                                                           |
|---------------------|------------------------------------------------------------------|
| `enter`             | Sessions: `claude --resume <uuid>`. Memories: open in `$EDITOR`. |
| `ctrl-r`            | Rename (textinput rendered at the row position)                  |
| `ctrl-x`            | Delete (two-step: select with `tab`, confirm with `enter`)       |
| `ctrl-s`            | Cycle sort field                                                 |
| `ctrl-/`            | Cycle layout: up -> right -> down -> left -> hidden              |
| `ctrl-o`            | Pager mode (`less`-style full preview)                           |
| `ctrl-u/d`          | Scroll preview                                                   |
| `/`                 | Search                                                           |
| `tab` / `shift-tab` | In delete mode: toggle selection. In search: cycle scope.        |
| `esc` / `q`         | Quit, or back-one-step in modal states                           |

### Memory type colors

| Type        | Color  |
|-------------|--------|
| `user`      | cyan   |
| `feedback`  | yellow |
| `project`   | green  |
| `reference` | blue   |

### Markdown rendering

Memory previews use Glamour for rendered markdown. The frontmatter shows at the top in plain text and the body is rendered below.

## Editor integration

The primary action for a memory invokes `$EDITOR` via `tea.ExecProcess`. This is the same pattern as `git commit` or `crontab -e`. The user gets their full editor (nvim or whatever) with their own config, plugins, and keybinds. Claudex pauses while the editor runs and resumes when it exits. There is no built-in editor.

## MEMORY.md index sync

Rename and delete operations rewrite the MEMORY.md index via regex line replacement. This preserves any surrounding formatting in the file. The pattern is roughly:

```go
linePattern := regexp.MustCompile(`(?m)^- \[([^\]]+)\]\(` +
    regexp.QuoteMeta(oldFilename) + `\).*$`)
```

If the regex doesn't match an existing entry during rename, we log a warning and skip the index update; the user can reconcile manually. The `.md` rename is the source of truth and `MEMORY.md` is metadata.

For concurrency, read mtime before the write, do the write, and retry once if mtime changed mid-operation. `flock` is held in reserve in case real conflicts appear in practice.

## Logging

Logging is the channel for things the user can't see in the UI. That means background goroutine failures (watcher errors, indexer parse skips, MEMORY.md regex mismatches), retried writes that gave up, and gated debug traces during development. User-visible problems belong in the UI, not the log.

The codebase uses `log/slog` from the standard library, calling `slog.Info`, `slog.Warn`, `slog.Error`, and `slog.Debug` directly. The `internal/logging` package owns the setup. `Init()` opens the log file and configures `slog.SetDefault`, and `Close()` releases the file on shutdown. Only `main` imports `internal/logging`; every other package uses slog's default logger.

The log file is at `$XDG_STATE_HOME/claudex/claudex.log`, falling back to `~/.local/state/claudex/claudex.log` if `XDG_STATE_HOME` is unset. This is the XDG state directory rather than the cache directory because the file is meant to survive across runs. The default level is `Info`; setting `CLAUDEX_LOG=debug` switches it to `Debug`. There is no log rotation in v1. Something like `lumberjack` can be added when the file actually grows unwieldy.

One constraint worth flagging: never write to stdout or stderr while `tea.Program.Run()` is active. The terminal belongs to the TUI during that time, and any direct writes will corrupt the rendered output. All logging during the run goes to the file. The pre-`Run()` setup phase and the post-`Run()` error path can write to stderr safely.

## Testing

TDD via `charmbracelet/x/exp/teatest`. Tests are organized as:

- Unit tests covering parsing (every JSONL message type plus malformed edge cases), title resolution, token and duration aggregation, frontmatter parsing, MEMORY.md line operations, column sizing, project-path decoding, sort behavior, and search-query translation.
- SQLite integration tests covering schema init, trigger firing, full and incremental parse, rewind detection, FTS5 stemming and ranking, and cross-search.
- Component tests via teatest covering tab switching, mode transitions, search flow, rename and delete, layout cycling, entity-change refresh, external editor handoff, and resize handling.
- Filesystem tests in temp dirs covering watcher debouncing, MEMORY.md exclusion, and end-to-end "create a file and confirm it appears in the list".

## Dependencies

| Package                                  | Purpose                                  |
|------------------------------------------|------------------------------------------|
| `charm.land/bubbletea/v2`                | TUI framework                            |
| `github.com/charmbracelet/bubbles`       | textinput, viewport, spinner, list, help |
| `github.com/charmbracelet/lipgloss`      | Styling and layout                       |
| `github.com/charmbracelet/glamour`       | Markdown rendering for memory preview    |
| `github.com/charmbracelet/x/exp/teatest` | TEA component testing                    |
| `modernc.org/sqlite`                     | Pure-Go SQLite with FTS5                 |
| `github.com/fsnotify/fsnotify`           | Filesystem watching                      |
| `github.com/sahilm/fuzzy`                | Fuzzy matching                           |
| `gopkg.in/yaml.v3`                       | Memory frontmatter parsing               |

If Bubbles, teatest, or Glamour turn out not to support Bubbletea v2 yet, we can pin Bubbletea back to v1. That decision waits until we actually hit the friction.

## Out of scope for v1

- Mouse support.
- Persistent preferences (active tab, sort order, last selection per tab, layout).
- Subagent conversation viewing (the `<uuid>/subagents/` subtree).

## Known limitations

- Project-path decoding is lossy (the `//` to `/.` heuristic): paths with literal consecutive dashes in directory names decode incorrectly.
- Cross-corpus BM25 ranking is approximate when using `UNION ALL`.
- Byte-offset incremental parse assumes JSONL is append-only; truncation triggers a full re-parse, which the sync table handles.
