---
name: template-sync
description: Updates this project from the template it was created from by porting new template commits. Use when asked to update, sync or pull changes from the template, or when `.template.json` is mentioned.
---

# Template Sync

If `.template.json` exists, this project was created from the template at `url`, and `commit` is the latest template commit whose changes have been applied. Without `.template.json`, there is no template to sync from.

## Steps

1. Read `.template.json`, and clone the template from `url` into a temporary directory.
2. List the template commits to port, oldest first, with `git log --reverse <commit>..main`.
3. Port each commit to this project, adapting names, URLs and paths. Skip:
   - Dependency updates (`chore(deps)`) that this project's Renovate already handles.
   - Changes to `init.sh`, `init.test.sh`, `template.schema.json`, the `templates` flake output and the `init` check, which are not part of this project.
   - If `library` is `true`, this project is a library: changes that only apply to applications, such as to the program, its container image, release artifacts and installation instructions.
4. Set `commit` to the template commit that was synced to in the same commit that applies its changes.
