# rexp

Named project workspaces for [Rex](https://www.superlogical.com/rex).

## Status

The source version is currently 1.0.0. `new` creates starter project configurations, `save` captures an existing Rex workspace, `list` shows saved project names, `validate` checks their structure, and `delete` removes saved configurations. Help, version output, and argument parsing are available. `start` launches the configured workspace through the installed Rex CLI, or selects an existing project session. `edit` opens saved configurations in an external editor and validates them after it exits. `stop` closes a project session while keeping its configuration. `status` reports its workspace session state. `search` finds saved names, `list --long`/`--json` expose configuration metadata, and `completion` generates shell integration.

On startup, rexp creates `.rexp/projects` inside the current user's home directory:

| Platform | Default directory |
| --- | --- |
| macOS | `/Users/<username>/.rexp` |
| Linux | Usually `/home/<username>/.rexp` |
| Windows | Usually `C:\Users\<username>\.rexp` |

The actual path comes from Go's `os.UserHomeDir` (`HOME` on macOS/Linux, `USERPROFILE` on Windows). The setup also runs for help and version invocations. Completion script generation and Tab-completion probes skip initialization and terminal status reports. Repeated runs preserve existing files and restore missing directories. On Unix, newly created directories use owner-only permissions. Initialization does not create configuration files yet. `XDG_CONFIG_HOME` does not change this location.

## Installation

See [INSTALLATION.MD](INSTALLATION.MD) for Go installation, prebuilt downloads, source builds, PATH setup, and shell completion.

## Development

Requires Go 1.25 or newer. Download dependencies and complete any missing `go.sum` entries first:

```sh
go mod tidy
go run . --help
go run . --version
go run . new --help
```

`go run . --version` prints `rexp 1.0.0`.

Build and check:

```sh
make build
make check
make test
./bin/rexp --help
```

The module path is `github.com/itsjustaplant/rexp`. Once a compatible version is published, install with `go install github.com/itsjustaplant/rexp@latest`; see [INSTALLATION.MD](INSTALLATION.MD) for PATH setup and other installation methods.

## Shell completion

Rexp uses [Cobra's shell completion](https://cobra.dev/docs/how-to-guides/shell-completion/) to generate scripts for Bash, Zsh, Fish, and PowerShell. Scripts complete subcommands and flags; saved project names are suggested for start, stop, status, edit, validate, delete, and search. Start/stop/status/edit/validate also suggest `.` when a regular `.rexp.yaml` exists in the current directory. `--file` completes YAML filenames and directories. Project names use case-sensitive prefix matching during completion, according to what you type; search uses case-insensitive substring matching.

Make sure the built binary is on PATH. From this repository, you can use `export PATH="$PWD/bin:$PATH"` in your current shell after `make build`.

Load completion for your current shell session:

```sh
# Zsh: initialize completion first if your shell setup has not already done so.
autoload -Uz compinit
compinit
source <(rexp completion zsh)

# Bash (with bash-completion available in your shell setup):
source <(rexp completion bash)

# Fish:
rexp completion fish | source
```

In PowerShell:

```powershell
rexp completion powershell | Out-String | Invoke-Expression
```

For example, type `rexp sta` then press Tab to complete `start`; type `rexp start my` then Tab to suggest matching saved projects. Tab does not execute the command.

To persist completion, add the appropriate load command to your shell startup file yourself (after your PATH and completion setup), or save a generated script and source it there. For Fish, save it to `~/.config/fish/completions/rexp.fish` after creating that directory. Rexp prints scripts to stdout; it never edits your shell config or installs scripts. Generated scripts query the current binary, so newly saved project names appear without regenerating the script. Regenerate/reload after updating the CLI's commands/flags.

Tab probes read filenames from `~/.rexp/projects` without parsing project YAML, contacting Rex, opening editors, or executing commands/hooks. Missing projects directories are treated as empty. Invalid YAML files can still be suggested by their valid names; invalid names, symlinks, directories, and unrelated files are ignored. Completion does not initialize `~/.rexp` or send program status notifications. Rex session IDs/client labels are not queried for suggestions.

## Search and detailed listing

```sh
rexp list
rexp search api
rexp search API --long
rexp list --long
rexp list --json
rexp search api --json
```

Plain `list` keeps its alphabetical, one-name-per-line output. Search matches a case-insensitive substring of the global saved **project name**, returning the same order. YAML contents are not searched. A missing/blank query is an error; no matches prints `No projects matched.` and succeeds. Repository-local configs are not registered or discovered by these commands.

`--long` (or `-l`) shows NAME, ROOT, TABS, PANES, STATUS, PATH, and ERROR columns. Counts include every tab and terminal pane across nested splits. STATUS means configuration validity, not whether a Rex session is running or its programs are healthy. Missing root directories do not invalidate a structurally valid config.

`--json` returns an array of objects with the same metadata. A valid entry includes `name`, `path`, `valid: true`, `root`, `tabs`, and `panes`. Invalid entries include `name`, `path`, `valid: false`, and `error`, omitting untrusted root/count metadata. An empty list or search result is `[]`. `--long` and `--json` are mutually exclusive.

Both detailed formats preserve malformed/unreadable configs as invalid entries and continue listing other projects. Entry-level validation failures do not make the command fail; errors reading the projects directory do. Plain list/search read filenames only. Detailed output parses selected files but never launches Rex or project code, and does not dump commands or environment values. The JSON fields are configuration metadata only.

## Terminal program status

Rexp reports its own lifecycle using [Mitchell Hashimoto's Program Status Protocol (OSC 7501)](https://www.superlogical.com/rex/docs/build/program-status):

| State | When rexp reports it |
| --- | --- |
| `working` | Executing a command or validating an edited file. |
| `blocked`, `kind=permission` | Waiting for deletion confirmation. |
| `blocked`, `kind=question` | Waiting for the external editor to exit. |
| `done` | A command completed successfully. This is the protocol's success state. |
| `error` | Argument parsing, initialization, or command execution failed. |
| `idle` | Deletion was declined or the command was interrupted. |

Reports use `app=rexp` and the named record `id=rexp` in the invoking terminal. This keeps them separate from reports made by editors and other programs. `start` completing means the workspace setup finished; it does not establish the health of programs launched in other panes. Successful saves print only `Saved <path>`; export limitations are documented below.

Status sequences are written only when stderr is a terminal. Stdout remains ordinary command output, and redirecting stderr to a file or pipe suppresses the sequences. Successful help and version invocations emit no status. `TERM=dumb` and `REXP_NO_STATUS=1` also disable reporting:

```sh
REXP_NO_STATUS=1 rexp list
rexp list 2>errors.log
```

Rexp does not query terminal input for protocol support or ring a bell. Terminals that support OSC 7501 decide how to display status and whether to notify or play sounds; other terminals can ignore the reports. No heartbeat or artificial delay is used. Short commands may finish too quickly to show a working indicator; deletion prompts and editor sessions provide longer-lived blocked states. Ctrl+C returns exit status 130 and leaves a cancelled deletion's configuration intact.

## Create a project

Create a named configuration from the project's directory:

```sh
cd ~/projects/my-app
rexp new my-app
```

This creates `~/.rexp/projects/my-app.yaml`, recording schema version 1, the current absolute directory as `root`, and a starter tab with a single `layout.pane`. Names must start with an ASCII letter or digit and contain at most 64 ASCII letters, digits, hyphens, or underscores. Windows-reserved names are rejected on all platforms. Existing configurations are never overwritten. New files use owner-only permissions on Unix.

`--edit` (or `-e`) opens the new file in your selected editor, waits for it to exit, and validates the result. The file remains available if editor launch or validation fails.

```sh
rexp new my-app --edit
rexp new my-app -e --editor "code --wait"
```

`--editor` on `new` requires `--edit`.

## Repository-local configurations

Keep a shareable configuration beside your code:

```sh
cd ~/projects/my-app
rexp new my-app --local
rexp edit .
rexp validate .
rexp start .
```

`new --local` creates `.rexp.yaml` with `root: "."`. It supports `--edit` and never overwrites an existing file. Commit this configuration to your project's repository if you want to share it. Local configurations are not registered in `~/.rexp/projects`, listed by `list`, or removed by `delete`.

`start`, `stop`, `status`, `validate`, and `edit` also accept `--file <path>`, including from another directory:

```sh
rexp start --file ~/projects/my-app/.rexp.yaml
rexp validate --file "./config/workspace file.yaml"
rexp edit --file ~/projects/my-app/.rexp.yaml --editor nvim
```

Choose a name, `.`, or `--file`; a name plus `--file` is an error. A bare name always selects the global named configuration. There is no automatic file discovery. Explicit file configs resolve relative or omitted `root` beside the configuration file, then resolve pane `cwd` against that root. Global named configs still require an absolute root. Files must be regular files; symbolic link files are rejected. Local sessions include a short hash of the canonical configuration path in their label, keeping separate checkouts and global projects independent. Moving a configuration file gives it a new session identity.

Reading or editing a configuration does not execute its commands or hooks. Starting a file explicitly executes its configured programs.

## Stop a workspace

```sh
rexp stop my-app
rexp stop .
rexp stop --file ~/projects/my-app/.rexp.yaml
rexp stop my-app --yes
```

Stop finds the same global or local session identity used by start, asks for confirmation, and closes the entire session through Rex. This includes all tabs, panes, running programs, and panes you added manually. Your project directory and YAML configuration are kept. `--yes` (or `-y`) skips confirmation. Closing the session containing the invoking terminal requires `--yes`; run stop from another session to use the normal prompt and see its final result.

A missing session prints `Project "my-app" is not running.` and succeeds without prompting. Incomplete `:starting` sessions can also be stopped. Duplicate matches, case-conflicting labels, and a replacement session appearing during confirmation produce an error instead of guessing. The selected session ID and label are rechecked before closure. Renamed sessions that no longer carry their workspace label are not selected. Stop requires a readable, valid config but does not require its root/pane directories to still exist.

An existing `after_close` observer handles the destruction event, just as when closing in Rex. Stop does not execute hooks itself, create an observer for reused sessions, or wait for background cleanup to finish. The documented close-hook limitations still apply. `before_close` remains unsupported. A failed Rex kill is reported and is not retried automatically; inspect the session before retrying.

## Workspace status

```sh
rexp status my-app
rexp status .
rexp status --file ~/projects/my-app/.rexp.yaml
rexp status my-app --json
```

Status loads the selected config and reads a single Rex session-list snapshot. It uses the same global/local identity as start and stop; a config with relative paths resolves beside the file. Pane/root directories do not need to exist. A valid config is required. It does not select, attach to, create, close, or modify sessions, inspect program health, or execute hooks/commands.

| JSON status | Meaning |
| --- | --- |
| `running` | Exactly one session has the expected workspace label. |
| `missing` | No matching session exists. Human output says the project is not running. |
| `incomplete` | Exactly one session has the expected `:starting` label. |
| `ambiguous` | Multiple sessions match, including a ready/incomplete pair, or a label differs only in case. Matching IDs/labels are shown without choosing one. |

A running session can contain exited programs or manual layout changes. Status does not compare the live layout to the config or establish whether services are ready. Renamed sessions that no longer match the workspace label are not tracked. Results describe the moment of the query and may change afterward.

All four observed states return exit status 0. Invalid arguments/configs, unreachable Rex, malformed responses, and cancellation return errors instead of being reported as missing. Use `--json` and inspect `status` in scripts. The JSON object contains `project`, `expected_label`, `status`, and `sessions` (an array of `session_id`/`label` objects, empty for missing); `reason` is included for ambiguity. It does not include commands or environment values. A missing named configuration is an error, distinct from a valid configuration with a missing session.

For an incomplete workspace, inspect it and use the matching `rexp stop` target before starting again. For ambiguous matches, inspect and rename duplicates/case conflicts in Rex. Status performs no recovery automatically.

## Lifecycle hooks

Project-level hooks are ordered argv arrays, like pane `commands`:

```yaml
hooks:
  before_start:
    - [./scripts/bootstrap]
  after_start:
    - [./scripts/workspace-ready]
  after_close:
    - [./scripts/cleanup]
```

- `before_start` runs before creating a new session. It can prepare pane directories before they are checked.
- `after_start` runs after tabs, panes, and initial focus are configured, before the session is marked ready. It does not wait for services in panes to become healthy.
- `after_close` runs after the entire session is destroyed, including through Rex's UI or `rex kill`. Closing one pane/tab or merely switching away does not trigger it while the session remains alive.

Hooks execute on the machine running rexp, in the project root, with the invoking environment plus project `env`. Tab/pane overrides do not apply. `REXP_PROJECT` identifies the project; `REXP_SESSION_ID` is empty before creation and contains the actual session ID for later phases. These metadata variables take precedence over configured values. Arguments remain literal; use an explicit shell for pipes, redirection, or expansion. Hook processes have closed stdin. Each command waits for the previous one and failure stops that phase. Shell-local changes do not carry into the next command or pane.

Reusing a session runs no hooks and creates no additional close observer. Startup hook output goes to the invoking terminal. Before-start failure prevents creation; later failure preserves the session with its `:starting` label for inspection. Hooks are not automatically retried. Ctrl+C cancels startup. Existing sessions that were not started with a close observer do not acquire one on reuse.

Rexp uses [Rex's session-destroyed event](https://www.superlogical.com/rex/docs/reference/events) for `after_close`. A detached worker subscribes before startup is marked ready; subscription failure fails startup. Its private snapshot and output log live under `~/.rexp/state/hooks/` (directories 0700, files 0600 on Unix). The snapshot includes the configured project env and close commands; it is removed when the observer ends, while the `.json.log` remains for inspection. Close hooks use the startup snapshot even if you later edit the config. The background hook has a five-minute timeout, and writes output/errors to its log without terminal status notifications.

`before_close` is **not implemented**: Rex provides an event after destruction, rather than a documented interception point. A nonempty `hooks.before_close` is rejected. Close hooks are best effort, not durable jobs: server disconnections, observer termination, crashes, and machine shutdown do not guarantee execution. EOF/disconnect alone never triggers cleanup, and observers are not restarted or reattached automatically. Save cannot infer hooks from a live session.

## Edit a project

```sh
rexp edit my-app
rexp edit my-app --editor nvim
rexp edit my-app --editor "code --wait"
```

The configuration must already exist. Invalid YAML can still be opened so you can repair it. Rexp waits for the launched editor process, then validates the saved configuration. Editor failures and invalid saves return errors; edits are preserved. Editing makes no Rex connection and does not modify an existing workspace. A reused session keeps its previous layout; changes apply when creating a new session.

Editor selection uses the first available setting in this order:

1. Explicit `--editor` option.
2. `editor` in `~/.rexp/config.yaml`.
3. `$VISUAL`.
4. `$EDITOR`.
5. `vi` on macOS/Linux, or `notepad.exe` on Windows.

The tool configuration is optional and is not automatically created. For example:

```yaml
# ~/.rexp/config.yaml
editor: code --wait
```

An empty setting falls through to the next choice. A malformed settings file or malformed selected editor command produces an error. An explicit `--editor` bypasses the settings file, allowing you to edit a project while fixing tool preferences separately. Commands that do not open an editor do not read these preferences.

Commands can include arguments. Quote an executable path or argument containing spaces, for example `editor: '"/path/to/My Editor/bin/editor" --wait'`. Single/double quotes and escaped quotes/spaces are supported; backslashes before ordinary characters remain literal for Windows paths. Rexp launches the program directly: shell variables, `~`, aliases, pipes, redirection, and command substitution are not evaluated. Use an absolute path or a program on `PATH`.

For GUI editors, include their wait option, such as `code --wait`; otherwise their launcher may exit before you finish editing. Terminal editors use the current terminal's input and output.

## Save a running workspace

Run inside the Rex session you want to capture:

```sh
rexp save my-app
```

This creates `~/.rexp/projects/my-app.yaml` from the session identified by `REX_SESSION`, which is the session containing your terminal, even if the app is displaying another session. To choose explicitly, including when running outside Rex:

```sh
rexp save my-app --session <rex-session-id-or-label>
```

The current directory becomes the configuration's `root`. Each pane gets its observed absolute directory: foreground process first, then the nearest readable ancestor, then the terminal's child process. If no process directory is available, save uses the last reported terminal directory. If no absolute directory can be obtained, saving fails rather than substituting an unrelated path.

Save preserves tab order, tab labels, nested horizontal/vertical splits, and each tab's focused pane. Split fractions are rounded to integer percentages summing to 100; very narrow splits that would round to 0 or 100 are rejected. Invalid or duplicate pane labels are given valid unique identifiers, and blank/duplicate tab names are normalized. Save records the active tab as `selected: true`, retaining tab order and each tab's pane focus.

**Rex does not expose complete process arguments.** Saved panes open interactive shells; add `command` arrays manually before starting if you want to launch programs. Saving does not capture terminal scrollback, shell history, environment changes, editor buffers, or running process state. It does not replay commands inferred from a title or process name.

Only terminal panes in one tiled layer per tab are supported. Floating/multiple layers, other block types, and panes on different hosts are rejected. Paths are native to the machine running rexp; remote workspace restoration is outside the current scope. The session is inspected again before writing, and changes to its layout, labels, or focus cause a retry error. Directory observations are collected sequentially, not as an atomic process snapshot.

Existing files, directories, and symbolic links at the destination are never overwritten. Use a new project name to save another snapshot. Saving only reads Rex state and writes the new configuration; it does not rename, close, or otherwise modify the source session.

```sh
rexp save saved-workspace
rexp validate saved-workspace
rexp start saved-workspace
```

Starting creates a session named `rexp:saved-workspace`, or reuses one already carrying that label. Use a fresh name when testing restoration so reuse does not hide changes.

## Layout configuration

```yaml
schema_version: 1
name: my-app
root: /absolute/path/to/my-app
tabs:
  - name: Development
    focus: editor
    layout:
      split: columns
      sizes: [50, 50]
      children:
        - pane: editor
          command: [nvim, .]
        - split: rows
          sizes: [34, 33, 33]
          children:
            - pane: server
              cwd: ./backend
              command: [npm, run, dev]
            - pane: logs
              command: [tail, -f, ./logs/app.log]
            - pane: shell
              command: []
```

`columns` arranges children left to right and divides width; `rows` arranges them top to bottom and divides height. `sizes` contains positive integer percentages summing to 100, one per child. Omit `sizes` for equal shares. Groups may nest, with at least two children per split. Each pane fills its parent's other dimension. Rex's CLI accepts integer split percentages, so some proportions are approximate; terminal cells and minimum pane sizes can also affect the result. Layouts requiring ratios outside Rex's 1–99 range are rejected before creation.

Each layout node defines either `pane` or `split`. Split nodes accept only `split`, `sizes`, and `children`; panes accept only `pane`, `command`, `commands`, `env`, and `cwd`. Pane identifiers follow project-name rules and must be unique across the project. Tab names must be nonempty and unique. `focus` must name a pane within its tab; omit it to focus the first pane. `selected: true` on a tab chooses the initial tab; at most one tab may be selected, and omitting it defaults to the first tab. Reusing an existing session preserves its current layout and focus.

Pane `cwd` defaults to `root`, and relative values resolve against `root`. Root must be absolute; use native paths for your OS. `command` is an array of strings; omitted or empty opens an interactive shell. Shell syntax needs an explicit shell, such as `[sh, -c, "npm run build && npm test"]`.

## Multiple commands in one pane

Use `commands` for an ordered sequence of argument arrays; `command` remains the single-program form. The two fields are mutually exclusive, including empty lists.

```yaml
pane: server
commands:
  - [npm, run, build]
  - [npm, run, dev]
```

Every command must contain a nonempty program name and string arguments. Commands run sequentially in the pane's directory, stopping on the first failure. A long-running command holds up later commands until it exits. A completed sequence exits with status 0; a failing command's nonzero status is preserved. Output stays visible through Rex's keep-open behavior. No shell is started after a finite sequence. An omitted/empty command list opens a shell normally.

Each entry is a separate process. Shell-local `cd`, `export`, and environment activation do not carry into subsequent entries. Use an explicit shell or a script if you need one shared shell context. Arguments are never implicitly interpreted as shell code.

Sequences use a private runner in the same built rexp executable, launched inside the terminal. Use a built/installed binary that remains available at its absolute path on the local Rex host; avoid `go run . start` for sequence workspaces because its temporary executable can be removed. The runner does not initialize `.rexp` or emit rexp lifecycle statuses in application panes. Ctrl+C stops the sequence before later commands. Saving a live session still cannot reconstruct full command arguments/sequences.

See [the combined workspace example](examples/workspace-features.yaml) for tab selection, sequences, scoped environment, and failure stopping together.

## Environment configuration

Set string-valued `env` maps on the project, a tab, or a pane. Values merge in this order: Rex's inherited launch/login environment → project → tab → pane. More specific values override broader values, including an explicitly empty string. Split groups do not accept `env`.

```yaml
env:
  NODE_ENV: development
tabs:
  - name: Server
    selected: true
    env:
      PORT: "3000"
    layout:
      pane: server
      env:
        DEBUG: "app:*"
      command: [npm, run, dev]
```

Names must match `[A-Za-z_][A-Za-z0-9_]*`; values must be strings without NUL. Quote numeric and boolean-looking values. Maps with duplicate names are rejected; names merge case-insensitively on Windows and case-sensitively on Unix. Null/unset directives, `.env` loading, and variable expansion are not implemented. `$NAME`, `${NAME}`, and shell-looking expressions in values remain literal.

Overrides are applied inside the terminal through the same private runner used for command sequences, after Rex establishes its login environment. Only explicit overrides are transported: inherited variables, such as the pane's Rex IDs, remain those provided by Rex unless explicitly overridden. All entries in a sequence receive the same final environment; command-side changes do not change the next process or the calling shell. Program lookup honors the final `PATH` (and `PATHEXT` on Windows); implicit execution from relative PATH entries is rejected in line with Go's `ErrDot` protection. Use explicit `./program` when intended.

A shell-only pane with overrides starts the inherited/configured `SHELL`, falling back to `/bin/sh` on Unix, or `COMSPEC`/`cmd.exe` on Windows. Interactive shell startup files can subsequently modify the environment. Panes without overrides retain their normal Rex shell behavior. Use a built/installed rexp binary available on the local Rex host for environment overrides as well as sequences.

Save records active-tab selection and layout, but it does not inspect/dump process environments or reconstruct command sequences; add the needed `env` and command fields manually to saved snapshots. Configuration files intentionally edited or serialized with these fields retain them.

## Validate a configuration

```sh
rexp validate my-app
```

Checks YAML syntax, schema version, names, node types, percentages, tab selection, focus references, command sequences, and environment maps. Unknown fields, duplicate YAML keys, aliases, null values, and multiple YAML documents are rejected. Layout nesting is limited to 32 levels. No commands run, and no Rex connection is made. This checks configuration structure, not whether paths/programs exist or the Rex server supports a particular layout.

See [the four-pane example](examples/four-panes.yaml). Existing files created with the previous `tabs[].panes` template are not automatically rewritten: add `schema_version: 1` and replace each tab's `panes` with a `layout` tree. `list` and `delete` still work on those filenames without parsing them.

## List projects

```sh
rexp list
```

Prints saved project names alphabetically (case-sensitive order), one per line, without the `.yaml` extension. An empty list shows how to create a project. Listing uses regular `.yaml` filenames with valid project names; other files, directories, and symbolic links are ignored. YAML contents are not read or validated, so a manually edited invalid YAML file still appears in the list. No Rex connection is required.

## Delete a project configuration

```sh
rexp delete my-app
```

The command shows the configuration's path and asks for confirmation. Enter `y` or `yes` to delete it; other answers cancel. With no input available, the command returns an error and keeps the file. To skip confirmation:

```sh
rexp delete my-app --yes
# Short form:
rexp delete my-app -y
```

Only `~/.rexp/projects/my-app.yaml` is removed. The project's source directory, other configurations, and running Rex sessions are left alone. Missing configurations, invalid names, directories, and symbolic links are rejected. Deleted configurations are not moved to a trash folder.

## Start a workspace

Launch a saved configuration from any directory:

```sh
rexp start my-app
rexp start my-app --background
rexp start my-app --client <rex-client-id-or-label>
```

Requires the Rex CLI on `PATH` and a reachable Rex server. `start` validates the entire configuration and checks all configured directories before creating terminals. This initial implementation is intended for local workspaces; it inherits Rex's server selection environment (`REX_SERVER`). Directory checks run on the machine running rexp.

Each project uses a session named `rexp:<project>`. A subsequent `start` selects that session without rerunning commands or applying configuration edits. Duplicate labels, labels differing only in case, and incomplete startups return errors. Session labels identify workspaces: renaming the session breaks this association. Avoid running simultaneous starts for the same project.

Rexp creates tabs and nested splits using returned Rex IDs, passes commands as argument arrays, labels each pane, and sets each tab's configured focus. Commands run once when their panes are created. Rex owns the terminals after rexp exits. Panes remain open after commands exit so their output is visible; successfully creating a pane does not mean its application is healthy. There is no readiness checking or dependency ordering between pane commands.

Selecting a session in the app requires Rex Remote Control. With multiple clients, use `--client` (or Rex's `REX_CLIENT` environment). `--background` skips app selection; it cannot be combined with `--client`. A selection error can occur after successful setup: retrying reuses the completed session.

During creation, the session is named `rexp:<project>:starting`. If setup fails, it remains under that name with existing panes running. Further starts stop with an error rather than rerunning commands. Inspect the session in Rex; use `rexp stop <project>` (or the matching local/file target) before retrying, or finish it yourself and rename it to `rexp:<project>`. Rexp does not automatically delete sessions or undo commands after a failure.

For a layout-only smoke test, use [the four-shell example](examples/four-shells.yaml), set `root` to an existing absolute directory, and save it as `~/.rexp/projects/four-shells.yaml`:

```sh
rexp validate four-shells
rexp start four-shells
rexp start four-shells
```

Expect one pane covering the left half and three stacked panes covering the right half, with the left pane focused. The second start prints `Reused` and creates no additional panes. Use `rexp edit my-app` to update the saved configuration.

## Release builds

[GoReleaser](https://goreleaser.com/) builds amd64/arm64 binaries for macOS, Linux, and Windows using `.goreleaser.yaml`. Archives include the license, installation instructions, the README, changelog, and example configs, plus a SHA-256 checksum manifest. Release binaries receive the tag version through `main.version`; `go install` builds use the version in the tagged source.

The Release workflow can be run manually to build nonpublishing snapshot artifacts. Pushing an intentional `v*` version tag runs checks, requires the tag to match the source version, and creates a **draft** GitHub Release. Review the draft before publishing binary downloads. A public Git tag can already be discovered by `go install` independently of whether the GitHub Release is published.

For a local preview with [GoReleaser v2](https://goreleaser.com/install/) installed:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

Snapshot archives/checksums are written under ignored `dist/`. No release tag, push, or publication is needed to prepare a snapshot. Release signing/notarization and Homebrew packaging are not configured. Complete platform verification before the public release.

## License

Rexp is source-available under the [MIT License with Commons Clause v1.0](LICENSE). You may use it internally for commercial work, fork it, modify it, and redistribute it subject to the license terms. The Commons Clause restricts selling rexp or providing paid products/services whose value derives entirely or substantially from its functionality; its definition includes certain hosting and consulting/support fees. It is not an OSI-approved open-source license. See LICENSE for the full terms. Third-party dependencies retain their respective licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
