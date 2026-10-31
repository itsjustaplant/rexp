# Changelog

## 0.1.0 — Unreleased

- Add nested pane layout configuration and `validate` for checking saved YAML.
- Add `start` to launch Rex tabs, panes, and commands, with existing-session reuse.
- Add `start --background` and `--client` for controlling Rex app selection.
- Preserve partial workspaces after startup failures and prevent automatic reruns.

- Initialize the user's `.rexp/projects` directory on startup.
- Add `new` to create starter YAML configurations with name validation and overwrite protection.
- Add `delete` with confirmation and an optional `--yes` flag.
- Add `list` to show saved project names alphabetically without parsing YAML.
- Explain missing project names with usage and examples.
- Reserve `new --edit` for future editor integration; currently it creates the file and prints a notice.

## 0.0.1

- Initialize the Go project with a Cobra command structure.
- Add help and version output.
- Define placeholders for init, edit, start, and list.
- Add build, formatting, vet, and test commands with CI configuration.
