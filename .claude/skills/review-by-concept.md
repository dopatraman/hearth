---
name: review-by-concept
description: Review a Hearth PR's diff grouped by architectural concept (data layer, business logic, HTTP, UI, etc.). Lists each file's role in its concept and ends with a reverse index for file lookup. Pass the PR number as the only argument.
---

Produce a scannable review of a PR, grouped by **what architectural concept each change serves**, not by directory.

### Steps

1. Resolve the PR number. If the user passed one as the argument, use it. Otherwise `gh pr list` and ask which.
2. Fetch the diff:
   - `gh pr view <N> --json files --jq '.files[].path'` for the file list
   - `gh pr diff <N>` for the actual changes if you need to understand a file's role beyond its name
3. Enumerate every changed file.
4. Group files by architectural concept. Typical buckets in this project:
   - **Data layer** — SQL migrations, `server/queries/*.sql`, sqlc-generated `server/internal/db/*.go`
   - **Business logic** — service structs in `server/internal/<domain>/`, error sentinels, validation
   - **HTTP surface** — handlers in `server/internal/api/*.go`, route mounts in `server.go`, DTOs
   - **Deploy / infra** — `deploy/*`, `.github/workflows/*`, systemd / Caddy configs
   - **iOS state** — `@Observable` stores (`BacklogStore`, `GraveyardStore`, `AuthStore`)
   - **iOS API client** — `HearthAPI.swift`, Codable DTOs in `Models.swift`, Keychain
   - **iOS UI** — SwiftUI views, rows, sheets, `Palette.swift`
   - **Mechanical** — xcodegen regen of `project.pbxproj`, lockfile churn

   Invent new buckets when the PR introduces something that doesn't fit, and skip empty buckets.

5. Under each concept, list the files with a one-line note on each file's role in that concept. Keep notes tight — a single sentence or clause.

6. End with a **reverse index** — alphabetical table of every changed file, mapping to its concept and (if evident from the diff) workflow(s). This is the lookup the user actually uses when scanning the review.

### Output format

```
### A. By concept (architectural layers)

**1. <Concept Name>**
- `path/to/file.ext` — one-line role
- `path/to/other.ext` — one-line role

**2. <Next Concept>**
- ...

**Mechanical regen** (not conceptual)
- `apps/Hearth.xcodeproj/project.pbxproj` — xcodegen output

### B. Reverse index — look up any file

| File | Concept | Workflow(s) |
|---|---|---|
| `path/one` | <concept>  | A, B |
| `path/two` | <concept>  | — (tests) |
```

### Things to also do

- Call out anything **silently dropped** (a UX, a feature, a test) — e.g., if a slice adds a date picker only via one entry point and omits another, say so.
- When a file's role is ambiguous, say so rather than invent.
- Keep tests grouped under the concept they verify; mark them `— (tests)` in the Workflow column of the reverse index.
- Call out any file whose contribution is pure mechanical (regen, lockfile) and don't try to assign it a concept.

### Honesty

Prefer "I'm not sure what this file does — looks like X" over confident fabrication. The point of this review is to help the reader understand the change, not to perform expertise.
