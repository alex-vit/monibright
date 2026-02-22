---
name: release
description: Tag a new release, update README download link, commit, tag, and push with release notes
user-invocable: true
---

# Release

Create a new versioned release of monibright.

## Steps

1. **Determine next version.** Find the latest version tag:
   ```
   git tag --sort=-version:refname | head -1
   ```
   Increment the patch number (e.g. `v1.0.0` → `v1.0.1`, `v1.2.9` → `v1.2.10`). Present this as the default option in AskUserQuestion, with minor and major bumps as alternatives. The version MUST match `v<major>.<minor>.<patch>`.

2. **Update README download link.** In `README.md`, find the existing download link (pattern: `[Download v...](https://github.com/alex-vit/monibright/releases/tag/v...)`). Replace both the link title and URL with the new version. For example, for `v1.2.0`:
   - Title: `Download v1.2.0`
   - URL: `https://github.com/alex-vit/monibright/releases/tag/v1.2.0`

3. **Write release notes.** Run `git log --oneline <previous-tag>..HEAD` to see all commits since the last release. Write a short bulleted list of **user-facing changes only** (new features, bug fixes, visible improvements). Skip internal-only changes (refactoring, build tweaks, notes/ideas updates, debug logging). Each bullet should be a concise phrase, not a commit message copy-paste. Save the notes to a shell variable for step 5.

4. **Commit.** Stage only `README.md`. Commit with message: `Update download link to <version>`.

5. **Tag and push.** Create the tag, then push the commit and tag together: `git push origin master <tag>`.

6. **Wait for the release.** The GitHub Actions workflow creates the release when the tag is pushed. Wait for it to appear (`gh run list --limit 1 --workflow=Release` until completed), then update its description with the release notes from step 3:
   ```
   gh release edit <tag> --notes "<notes>"
   ```
