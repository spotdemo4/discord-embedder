# AGENTS.md

## Task Completion Requirements

- `nix fmt` must be used for formatting.
- `nix flake check` must pass before considering tasks completed. Intent-to-add (`git add -N`) new files so `nix flake check` can see them.

## Version Control Requirements

- Commit messages created by agents must follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).
- Commits created by agents must include an `Assisted-by: [tool name] ([primary model name and version])` Git commit trailer.
- Branch names created by agents must follow [Conventional Branch 1.0.0](https://conventionalbranch.org/v1.0.0/).
- In pull request descriptions and commit messages, put a closing keyword (`close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`, `resolved`) right before an issue reference only when the change should close that issue, and a reopening keyword (`reopen`, `reopens`, `reopened`) only when it should reopen it. Forgejo, Gitea and GitHub act on a closing keyword on merge wherever it appears, even in a negated sentence such as "this does not close #123", and Forgejo and Gitea act on a reopening keyword the same way. Refer to an issue that stays as it is as "Refs #123" or "#123 stays open".
