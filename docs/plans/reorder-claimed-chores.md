# Reorder claimed chores — plan

## Context

The Hearth refresh canvas has a Reorder artboard (`project/Reorder.dc.html`) specing how claimed chores on the Backlog should be rearrangeable: long-press a row (~380ms) to lift it, drag up or down, release to drop. The new order is global — every member of the shared `lists` entry sees the same sequence on their Backlog — and a dark toast "Order updated for everyone" appears to confirm the save. No Undo. Concurrent reorders resolve last-write-wins (lenient).

Today the Backlog sorts the Claimed section by `(overdue DESC, willCompleteBy ASC NULLS LAST, createdAt ASC)` with no user control. The `chores` table has no ordering column. There is no reorder endpoint. Clients already receive silent "something changed on this list" pushes that trigger a full `GET /chores` refetch — that's the sync model we reuse.

**Data-model choice.** Position lives as a nullable `claimed_order INTEGER` column on `chores`, matching the shape the schema already uses for `claimed_by` and `claimed_at` (set on claim, NULL otherwise). For one-list-per-item with household-sized lists (~5–20 claimed rows) and rare concurrent edits, an integer position column on the row with transactional renumbering is the widely accepted convention. A side table was considered and rejected because it would be inconsistent with how the rest of claim state is stored; LexoRank was rejected as overkill at this list size.

---

## Slice — Reorder claimed chores

**User-facing value**: on the Backlog, long-press any claimed chore, drag it up or down, release. The new order saves; a dark confirmation toast "Order updated for everyone" appears; every household member's device refetches and shows the same sequence within a couple of seconds.

**Branch base**: `main` at `199c93f` (just pulled — claim/complete notifications merged).
**Branch**: `reorder-claimed-chores`

One slice, one PR. Server + iOS ship together so the slice is user-facing on merge. TDD on the server side (new endpoint has unit tests first). iOS is view-level work verified in the simulator per the Build Plan's design-review standard.

### Checklist — Server (`server/...`)

- [ ] **Migration** `server/migrations/00006_add_chores_claimed_order.sql`
  - [ ] `ALTER TABLE chores ADD COLUMN claimed_order INTEGER`
  - [ ] Backfill: for every row with `claimed_at IS NOT NULL`, set a dense per-`list_id` rank ordered by `((will_complete_by IS NOT NULL AND will_complete_by < strftime('%Y-%m-%dT%H:%M:%fZ','now')) DESC, will_complete_by ASC NULLS LAST, created_at ASC)` — mirrors the current client sort so returning users see no visual jump.
  - [ ] `CREATE INDEX chores_list_claimed_order ON chores(list_id, claimed_order) WHERE claimed_order IS NOT NULL`
- [ ] **Queries** in `server/queries/chores.sql`
  - [ ] Add `claimed_order` to the SELECT column list in `BacklogForUser` and `ListChoresForList`. No ORDER BY change — client still groups and sorts.
  - [ ] `AssignNextClaimedOrder` — sets `claimed_order = COALESCE(MAX(claimed_order), 0) + 1` within a `list_id` for a given chore id.
  - [ ] `ClearClaimedOrder` — sets `claimed_order = NULL` for a given chore id.
  - [ ] `RewriteClaimedOrder` — takes a `list_id` and an ordered list of chore ids; writes `claimed_order = 1..N` for those and `N+1..` for any other chore in the same list with `claimed_at IS NOT NULL` that was omitted, ordered by their current `(claimed_order ASC NULLS LAST, created_at ASC)`.
- [ ] **Handler wiring** in `server/internal/api/chores.go`
  - [ ] `handleClaimChore` — after the existing claim UPDATE, call `AssignNextClaimedOrder` in the same `UoW.InTx`.
  - [ ] `handleReleaseClaim` and `handleCompleteChore` — call `ClearClaimedOrder` in the same transaction.
  - [ ] DTO (`toChoreDTO`) exposes `claimed_order` as snake_case, nullable.
- [ ] **New endpoint** `POST /chores/reorder`
  - [ ] Route registered in `server/internal/api/server.go` alongside the other `/chores/...` routes.
  - [ ] Body `{"order": ["<chore_id>", ...]}`.
  - [ ] Resolve `user_id` from session. 400 if body empty or ids duplicated.
  - [ ] Fetch rows for every id; 400 if any id is missing, has `state != 'open'`, has `claimed_at IS NULL`, or belongs to a different `list_id` than the rest.
  - [ ] 403 if the user is not in `list_memberships` for that `list_id`.
  - [ ] One transaction: `RewriteClaimedOrder(list_id, ids)` (lenient — omitted still-claimed chores append in relative order).
  - [ ] `s.Dispatcher.Dispatch(ctx, listID, actorID, "")` — silent push, same shape as `handleSetWillCompleteBy` at `chores.go:260`. Actor excluded by `TokensForList`.
  - [ ] Respond 200 with the full chore list for the user (same DTO shape as `GET /chores`).
- [ ] **Tests** (TDD — write first) in `server/internal/api/chores_reorder_test.go`
  - [ ] Happy path: 2 members, 3 claimed chores; POST reorder; assert response rows have `claimed_order` 1..3 in the submitted order; a separate GET from member B shows the same order.
  - [ ] Dispatcher recorded with `actor_id = memberA`, `alert = ""` (existing in-process `notifications.Pusher` fake from `push_test.go`).
  - [ ] Lenient append: B claims a 3rd chore after A's fetch; A reorders submitting only the 2 ids it knew; assert the 3rd lands at `claimed_order = 3`.
  - [ ] 403 for non-member.
  - [ ] 400 for an id that is unclaimed, completed, or belongs to a different list.
  - [ ] Claim assigns next order; release clears order; complete clears order (three small cases).
  - [ ] Migration backfill: insert fixtures pre-migration, run migration, assert dense 1..N ranks per list in the expected tiebreaker order.

### Checklist — iOS (`apps/HearthApp/...`)

- [ ] **Model** — `Models.swift`
  - [ ] Add `claimedOrder: Int?` to `HearthChore`, decoded from `claimed_order`.
  - [ ] Thread it through `CodingKeys` and any preview fixtures.
- [ ] **Sort** — `BacklogSections.swift` L98-113
  - [ ] Claimed-section sort becomes `claimedOrder ASC NULLS LAST`, with the current `(overdue, willCompleteBy, createdAt)` tuple as a tiebreaker for the brief window before the server has assigned orders. "Up for grabs" and "Snoozed" sorts untouched.
- [ ] **Store** — `BacklogStore.swift`
  - [ ] `func reorderClaimed(_ newOrder: [HearthChore.ID])`.
  - [ ] Snapshot current `chores` for rollback.
  - [ ] Reassign `claimedOrder` locally as 1..N for the submitted ids; publish immediately so the UI stays settled.
  - [ ] Call `api.reorderChores(order: newOrder.map(\.uuidString))`.
  - [ ] On success: replace local chores with the server response (picks up any lenient append from a concurrent claim).
  - [ ] On failure: restore the snapshot and set a toast like "Couldn't reorder".
- [ ] **API** — `HearthAPI.swift`
  - [ ] `func reorderChores(order: [String]) async throws -> [HearthChore]` POSTing to `/chores/reorder`.
- [ ] **Drag gesture** — `BacklogView.swift` L232-257
  - [ ] Replace the claimed section's static `VStack(ChoreRow)` with a hand-rolled reorderable stack. `STEP = 86` pt per row (10pt spacing + 76pt card).
  - [ ] `LongPressGesture(minimumDuration: 0.38).sequenced(before: DragGesture())` driving per-row `@State` for `isPressing` / `isLifted` / `dragOffset`.
  - [ ] On lift: `UIImpactFeedbackGenerator(style: .light)`, scale card to 1.035, rotate -0.6°, fade in a 6-dot grip SVG on the right, dim the "Up for grabs" section below to opacity 0.45.
  - [ ] During drag: translate the lifted card by `dragOffset.height`, clamp to list bounds. `toIndex = fromIndex + round(dy / STEP)`. Non-lifted rows between from/to shift ±STEP with `.spring(response: 0.28, dampingFraction: 0.78)` to open the gap.
  - [ ] On release: `UIImpactFeedbackGenerator(style: .medium)`, animate the dropped card into its slot with `.spring(response: 0.32, dampingFraction: 0.58)` (settle) plus a small overshoot bounce. Call `store.reorderClaimed(newOrder)` iff `newOrder != oldOrder`.
- [ ] **Hint** — `BacklogView.swift`
  - [ ] Small 12pt "Hold to rearrange" label in the Claimed section header, trailing-aligned. Opacity 1 until a successful lift, then 0.
  - [ ] Persist `hasUsedReorder` in `UserDefaults` so the hint stays gone across launches.
- [ ] **Confirmation toast** — `BacklogView.swift` L110-129 (reuse existing overlay)
  - [ ] On success, present `ToastModel(text: "Order updated for everyone", undoLabel: nil, kind: .success)`. `ToastView` already conditionally renders Undo, so `nil` yields a confirmation-only capsule.
  - [ ] Auto-dismiss at 3.2s, matching the existing completion toast timing.
  - [ ] Skip the toast and the network call when `newOrder == oldOrder`.
- [ ] **Push wiring** — no change. `RootTabView.swift` L48-50's existing `PushRegistration.onRefresh → backlog.reload()` already delivers the new order to the other member on the silent APNs fan-out.

### Test plan

Server:

1. `just test` green on a fresh clone; migration runner shows 00006 applied.
2. `go run ./cmd/hearth` locally; `sqlite3 ./hearth.dev.db ".schema chores"` shows the new column and index.
3. Curl smoke on localhost: sign in as member A, create and claim 3 chores, POST `/chores/reorder`, GET `/chores`, confirm new orders persist.

iOS, on two devices (or one device + one simulator) signed into two members of the same list, each with ≥3 claimed chores, against the deployed prod server:

4. `xcodebuild test` — existing iOS suite passes.
5. **Happy path**: on A, long-press a claimed chore, drag up, release. Card lifts with light haptic + scale 1.035 + tilt -0.6°, grip dots fade in, other rows slide to open a gap, "Up for grabs" dims to 0.45. On release: medium haptic, card settles with a springy land, confirmation toast fades in bottom-center and auto-dismisses at 3.2s.
6. **Persistence**: pull-to-refresh on A — order holds.
7. **Cross-member propagation**: within ~2s of A's drop, B's Backlog refetches and shows the same new order.
8. **Clamp**: drag past both ends — the lifted card stops at the ends; `toIndex` stays in bounds.
9. **Release without moving**: long-press, don't drag, release — no toast, no network call.
10. **Network loss**: airplane mode, drag, release — local order reverts; "Couldn't reorder" toast appears.
11. **Concurrent-claim lenient path**: on A, fetch. On B, claim a new chore. On A, reorder without touching B's new id. After A's server response, B's new chore sits at the bottom of A's Claimed section.
12. **Hint**: first launch after install shows "Hold to rearrange"; after one successful reorder, relaunch — the hint is gone.
13. **Design review** per the Build Plan standard: iPhone 13 mini, light and dark, record a short screen capture for the PR description.

---

## Critical files

Server:
- `server/migrations/00006_add_chores_claimed_order.sql` (new)
- `server/queries/chores.sql`
- `server/internal/api/chores.go`
- `server/internal/api/server.go`
- `server/internal/api/chores_reorder_test.go` (new)

iOS:
- `apps/HearthApp/Models.swift`
- `apps/HearthApp/BacklogSections.swift`
- `apps/HearthApp/BacklogStore.swift`
- `apps/HearthApp/HearthAPI.swift`
- `apps/HearthApp/BacklogView.swift`

## What's not in scope

- No change to `claimed_by` / `claimed_at` storage. If we later extract all claim state into a dedicated `claims` table, it's a separate refactor.
- No Undo. The Reorder artboard shows an Undo button; we drop it per your call.
- No LexoRank / fractional keys. Revisit only if the Claimed section ever gets long enough to feel the renumber cost.
- No watchOS Reorder. The artboard is iPhone only.
- No animation tuning for other section moves. Slice 23 handled claim/complete animations; we leave those alone.
