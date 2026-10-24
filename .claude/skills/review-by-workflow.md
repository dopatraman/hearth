---
name: review-by-workflow
description: Review a Hearth PR's diff grouped by user-facing workflow (what the user does end-to-end). Each workflow lists every participating file and its role. Ends with a reverse index. Pass the PR number as the only argument.
---

Produce a review of a PR grouped by **what the user does end-to-end** — not by architectural layer. Each workflow is a chain of files from UI gesture → store → API → HTTP handler → service → SQL.

### Steps

1. Resolve the PR number. If passed as argument, use it. Otherwise ask.
2. Fetch the diff:
   - `gh pr view <N> --json files --jq '.files[].path'`
   - `gh pr diff <N>` when needed to understand a specific role
3. Identify user-facing workflows the PR introduces or modifies. A workflow is a thing a user does end-to-end: "claim a chore", "accept an invite", "see that a chore is overdue". Not infrastructure refactors — those get a brief "cleanup / not a workflow" footnote.
4. For each workflow:
   - Write a one-sentence summary of what the user does end-to-end.
   - Follow with a markdown table of `(file, role)` pairs. List every file that participates — from the UI entry point down to the SQL. Order top-down (UI → store → API → handler → service → query → migration).
5. End with a **reverse index** mapping every file to its workflow(s) and (if conceptually distinct) its architectural concept.

### Output format

```
### A. By workflow (user-facing flows)

**Workflow A — <one-sentence summary of the user-facing action>**

| File | Role |
|---|---|
| `apps/HearthApp/<EntryView>.swift` | <role> |
| `apps/HearthApp/<Store>.swift` | <role> |
| `apps/HearthApp/HearthAPI.swift` | <role> |
| `server/internal/api/<handler>.go` | <role> |
| `server/internal/api/server.go` | Route mount |
| `server/internal/<domain>/service.go` | <role> |
| `server/internal/db/<generated>.go` | sqlc wrapper |
| `server/queries/<file>.sql` | <SQL purpose> |

**Workflow B — ...**

**Not a workflow** (cleanup / mechanical / infra)

| File | What |
|---|---|
| `apps/Hearth.xcodeproj/project.pbxproj` | xcodegen regen |
| `path/to/other` | <brief note> |

### B. Reverse index — look up any file

| File | Workflow(s) | Concept |
|---|---|---|
| `path/one` | A | <concept> |
| `path/two` | — (tests) | <concept> |
```

### Things to also do

- Call out anything **silently dropped**. If a PR adds UI for one entry point to a feature but not another (e.g., swipe-to-claim opens the date picker, but tapping an existing claimed row doesn't let you change the date), say so explicitly in a "Not implemented this slice" callout.
- When a workflow depends on a prior PR's groundwork (schema already exists, UI structure was in place), note that — don't list unchanged files, but mention them in prose so the reader knows the full picture.
- Tests go in the reverse index with `— (tests)` in the Workflow column.
- Mechanical regen gets its own "Not a workflow" bucket; don't try to shoehorn `project.pbxproj` into Workflow A.

### Honesty

Prefer "this file's role is unclear from the diff" over inventing a plausible-sounding story. If two workflows share a file (common for `server.go` route mounts), list it under both.
