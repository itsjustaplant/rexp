# Changelog

## 1.1.0 — 2026-11-03

- Bundle a shared workspace skill for Codex and Claude Code with `skill show`, `skill install`, and `skill remove` commands.
- Add `switch` to select running Rex sessions by project shorthand, exact label, or full ID, with local/file workspace targeting and explicit client selection.
- Add `use` to start a saved workspace if needed and switch to it, reusing existing sessions without rerunning programs or hooks.
- Add first-run interactive onboarding and replayable `init`/`intro` commands with optional skill installation and shell completion setup.
- Install completion for Zsh, Bash, and Fish with user consent, preserving existing settings and avoiding duplicate setup blocks.
- Include a two-pane welcome demo with a live ASCII clock and the rexp dinosaur logo.

## 1.0.0 — 2026-11-02

First release preparation. Publication and distribution remain separate steps.

- Create, edit, validate, list, search, and delete saved project configurations.
- Start, inspect status, stop, and save Rex workspaces with named and repository-local configs.
- Configure nested pane layouts, percentage sizes, tabs, focus/selection, ordered commands, and scoped environments.
- Run startup and whole-session after-close hooks, with terminal program-status reporting.
- Generate Bash, Zsh, Fish, and PowerShell completion; provide detailed and JSON project listing.
- Use the canonical `github.com/itsjustaplant/rexp` module path for Go installation.
- Add GoReleaser platform builds, checksums, nonpublishing snapshots, and draft GitHub releases.
- Document installation, PATH setup, and manual updates.
- License rexp under MIT with Commons Clause v1.0, permitting internal commercial use while restricting sale as defined in the license.

## 0.5.0 (development)

- Simplify successful save output to a single `Saved <path>` line.
- Add project/tab/pane environment overrides with scoped precedence, native name handling, and effective-PATH command lookup.
- Add sequential pane `commands` with literal argument arrays, failure stopping, and cancellation.
- Add initial tab selection with `selected: true`, including active-tab preservation in `save`.

## 0.4.0

- Report command lifecycle through OSC 7501: working, blocked for approval/editing, done, error, and idle on cancellation; suppress reports for redirected stderr and help/version.

## 0.3.0

- Add `edit` and editor launching for `new --edit`, with configurable editor selection, terminal input/output, and validation after editing.

## 0.2.0

- Add `save <project>` to capture Rex tabs, nested layouts, pane labels, focus, and directories into a new configuration; full command arguments remain manual.
- Add nested pane layout configuration and `validate` for checking saved YAML.
- Add `start` to launch Rex tabs, panes, and commands, with existing-session reuse.
- Add `start --background` and `--client` for controlling Rex app selection.
- Preserve partial workspaces after startup failures and prevent automatic reruns.

## 0.1.0

- Initialize the user's `.rexp/projects` directory on startup.
- Add `new` to create starter YAML configurations with name validation and overwrite protection.
- Add `delete` with confirmation and an optional `--yes` flag.
- Add `list` to show saved project names alphabetically without parsing YAML.
- Explain missing project names with usage and examples.
- Reserve `new --edit` for editor integration (implemented in 0.3.0).

## 0.0.1

- Initialize the Go project with a Cobra command structure.
- Add help and version output.
- Define placeholders for init, edit, start, and list.
- Add build, formatting, vet, and test commands with CI configuration.
