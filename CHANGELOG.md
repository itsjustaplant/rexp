# Changelog

## 0.4.0 — Unreleased

- Add initial tab selection with `selected: true`, including active-tab preservation in `save`.

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
