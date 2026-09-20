# Repository agent notes

## GitHub CLI from PowerShell

- PowerShell does not interpret `\n` inside a quoted string as a newline.
- For multiline `gh pr create` or `gh pr edit` bodies, use a PowerShell here-string (`@' ... '@`) or `--body-file` instead of embedding `\n` escapes.
