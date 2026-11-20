# Reorder unclaimed chores — plan

## Context

Slice `reorder-claimed-chores` (PR #43) shipped drag-to-reorder for the Backlog's Claimed section. The Up-for-grabs section is still sorted by `created_at ASC` with no user control. This slice extends the same long-press + drag interaction to Up-for-grabs so household members can prioritize what's available to be claimed, with the same global/last-write-wins semantics the Claimed section already has.

**Data-model choice.** Generalize the existing `claimed_order INTEGER` column on `chores` to `section_order INTEGER`. Meaning becomes "position within whichever section (Claimed or Up-for-grabs) the chore currently lives in." NULL when completed. One column, two sections:

| Chore state | `section_order` |
|---|---|
| open, claimed | position in Claimed (`1..K` within the list's claimed set) |
| open, unclaimed | position in Up-for-grabs (`1..K` within the list's unclaimed set) |
| completed | NULL |

**Section transitions move the chore to the bottom of the destination; the source section leaves a gap.** On claim, Up-for-grabs loses a slot and Claimed gains one at `MAX + 1`. On release, the mirror. No gap-closing — gaps are invisible to users (sort only cares about relative order), `AssignNextSectionOrder` keeps new arrivals out of any hole, and a drag in that section renumbers it back to dense `1..N` via `assignRanks`. The simpler "single-write section move" design we're already using for claim/release carries over verbatim.

Scope-wise the shape mirrors the first slice: one migration, one endpoint (unchanged path, broader validation), a parallel sort rule on iOS, and the hand-rolled drag view reused for a second section.

---

## Slice — Reorder unclaimed chores

**User-facing value**: on the Backlog, long-press any chore — claimed or Up-for-grabs — drag, release. The new order saves globally, every household member's device refetches, and the confirmation toast "Order updated for everyone" appears.

**Branch base**: `main` at the commit that merges PR #43 (`reorder-claimed-chores`).
**Branch**: `reorder-unclaimed-chores`

One slice, one PR. Server + iOS ship together. TDD on the server, simulator-verified on iOS.

### Checklist — Server (`server/...`)

- [ ] **Migration** `server/migrations/00007_rename_claimed_order_to_section_order.sql`
  - [ ] `ALTER TABLE chores RENAME COLUMN claimed_order TO section_order`
  - [ ] Backfill Up-for-grabs rows: for every `state = 'open' AND claimed_at IS NULL` row, write a dense per-`list_id` rank ordered by `created_at ASC`. Claimed rows already carry their values from slice 24's backfill.
  - [ ] Drop `chores_list_claimed_order` index; create `chores_list_section_order ON chores(list_id, section_order) WHERE section_order IS NOT NULL`.
- [ ] **Queries** in `server/queries/chores.sql`
  - [ ] Rename column references across every SELECT and RETURNING clause (`claimed_order` → `section_order`).
  - [ ] `AssignNextSectionOrder` (renamed from `AssignNextClaimedOrder`): the subquery becomes section-aware — compares `claimed_at IS NULL` to the target row's own `claimed_at IS NULL`, so MAX is scoped to the target's current section. Same shape otherwise.
  - [ ] `ClearSectionOrder` (renamed from `ClearClaimedOrder`).
  - [ ] `SetSectionOrder` (renamed from `SetClaimedOrder`).
- [ ] **Service** (`server/internal/chores/service.go`)
  - [ ] `Create` without claim: after insert, call `AssignNextSectionOrder` so the chore lands at the bottom of Up-for-grabs. (Currently chores created unclaimed carry `section_order = NULL` — this is the behavior change.)
  - [ ] `Create` with claim: unchanged flow — `ClaimChore` + `AssignNextSectionOrder` places it at the bottom of Claimed.
  - [ ] `Claim`: unchanged — `AssignNextSectionOrder` after `ClaimChore` naturally scopes MAX to the new section (now that claimed_at is non-null), overwriting the old Up-for-grabs value.
  - [ ] `Release`: after `ReleaseClaim` clears `claimed_at`, call `AssignNextSectionOrder` to place in Up-for-grabs at the bottom. Drop the explicit `ClearSectionOrder` call that was there for the pre-refactor Claimed-only model.
  - [ ] `Complete`: `ClearSectionOrder` still runs; sets to NULL.
  - [ ] `validateReorderTargets`: broaden to accept either "all claimed" or "all unclaimed," enforced consistent within the body. Add `ErrReorderMixedSections` for a body that mixes states. Keep same-list check.
- [ ] **Endpoint** `POST /chores/reorder`
  - [ ] Route and handler unchanged in shape. Validation layer now handles both sections. DTO exposes `section_order` instead of `claimed_order`.
- [ ] **Tests** (`server/internal/api/chores_reorder_test.go`)
  - [ ] Reorder Up-for-grabs happy path: three unclaimed chores, POST reorder, assert new section_order 1..3 in submitted order.
  - [ ] Mixed-section body → 400 `ErrReorderMixedSections`.
  - [ ] Create without claim assigns the next Up-for-grabs rank.
  - [ ] Claim moves a chore to Claimed bottom (next MAX+1 among claimed); old Up-for-grabs position becomes a gap.
  - [ ] Release moves a chore to Up-for-grabs bottom (next MAX+1 among unclaimed); old Claimed position becomes a gap.
  - [ ] Complete sets section_order to NULL (existing test, column rename only).
  - [ ] The existing `TestReorder_SilentPushToOtherMembers` and lenient-append tests continue to pass for the Claimed path.

### Checklist — iOS (`apps/HearthApp/...`)

- [ ] **Model** (`Models.swift`): rename `claimedOrder: Int?` → `sectionOrder: Int?` and the JSON key `claimed_order` → `section_order`. Rename `with(claimedOrder:)` helper.
- [ ] **Sort** (`BacklogSections.swift`): both the Claimed-section sort and the Up-for-grabs sort lead with `sectionOrder ASC NULLS LAST`, then the existing tiebreakers (overdue/willCompleteBy/createdAt for claimed; createdAt alone for unclaimed). Snoozed section keeps its current `createdAt` sort.
- [ ] **Store** (`BacklogStore.swift`): rename `reorderClaimed(_:)` → `reorderSection(_:)`. Logic unchanged — still snapshot, optimistically assign `sectionOrder` locally, POST, replace from response or rollback. `setReorderLifted(_:)` wiring from the refresh-suspension fix stays.
- [ ] **API** (`HearthAPI.swift`): `reorderChores(sessionToken:order:)` unchanged — endpoint path and body shape are the same.
- [ ] **View** (`ReorderableClaimedSection.swift`): rename the file + type to `ReorderableSection.swift` / `ReorderableSection`. No internal logic changes — the component is already generic over "an array of chores + a row builder."
- [ ] **View** (`BacklogView.swift`): wrap the Up-for-grabs section's row rendering with `ReorderableSection` the same way Claimed is wrapped. Pass `section.items`, the shared `hasUsedReorder` binding, the lift callback, and `store.reorderSection` as `onReorder`.
- [ ] **Hint** (`SectionHeader.swift` already supports a `hint` param): show "Hold to rearrange" on both Claimed and Up-for-grabs section headers while `!hasUsedReorder && items.count > 1`. First successful reorder in either section flips `hasUsedReorder` and the hint disappears from both.
- [ ] **Dim** (`BacklogView.swift`): keep the 0.45 dim behavior but make it bidirectional — whichever section isn't currently being dragged dims. `isReorderLifted` is already tracked; the other section just reads the same flag.

### Test plan

Server:

1. `just test` green on a fresh clone; migration runner shows 00007 applied; `sqlite3 ./hearth.dev.db ".schema chores"` shows the renamed column and index.
2. Curl smoke on localhost: sign in; create one chore unclaimed, confirm `section_order = 1` on the GET; create another; POST `/chores/reorder` with the two unclaimed ids reversed; GET confirms the swap.
3. Curl smoke: claim the first chore; GET confirms its `section_order` jumps to `1` in Claimed, while the second unclaimed chore still has `section_order = 2` in Up-for-grabs (gap). Release it; GET confirms it lands at the bottom of Up-for-grabs (`section_order = 3`).

iOS, on two devices (or one device + one simulator) signed into two members of the same list, against the deployed prod server:

4. `xcodebuild test` — existing 105 iOS tests pass.
5. **Happy path, unclaimed**: on A, long-press an Up-for-grabs chore, drag, release. Lift + slide + settle visuals match the Claimed section; confirmation toast appears.
6. **Cross-member propagation**: within ~2s, B's Backlog refetches and reflects the new Up-for-grabs order.
7. **Hint**: fresh install shows "Hold to rearrange" on both headers; after one reorder (in either section), hint is gone from both on relaunch.
8. **Dim**: lifting a Claimed chore dims Up-for-grabs; lifting an Up-for-grabs chore dims Claimed. Snoozed never dims.
9. **Section transitions**: on A, claim an Up-for-grabs chore — it jumps to the bottom of Claimed. Release — it returns to the bottom of Up-for-grabs (not its original Up-for-grabs slot, by design). The Snoozed section is unaffected.
10. **Mixed-section body**: via curl, submit a reorder body mixing claimed + unclaimed ids — assert 400.
11. **Design review** per the Build Plan standard: iPhone 13 mini, light + dark, record a short screen capture for the PR.

### Critical files

Server:
- `server/migrations/00007_rename_claimed_order_to_section_order.sql` (new)
- `server/queries/chores.sql`
- `server/internal/db/chores.sql.go` (sqlc regenerated)
- `server/internal/db/models.go` (sqlc regenerated)
- `server/internal/chores/service.go`
- `server/internal/api/chores.go` (DTO field rename)
- `server/internal/api/chores_reorder_test.go` (new tests + renames)

iOS:
- `apps/HearthApp/Models.swift`
- `apps/HearthApp/BacklogSections.swift`
- `apps/HearthApp/BacklogStore.swift`
- `apps/HearthApp/HearthAPI.swift`
- `apps/HearthApp/ReorderableClaimedSection.swift` → renamed to `apps/HearthApp/ReorderableSection.swift`
- `apps/HearthApp/BacklogView.swift`

## What's not in scope

- **No gap-closing on claim/release.** The source section leaves a hole at the moved chore's old slot. The hole heals the next time someone drags in that section. Decided explicitly after walking through the write cost (N vs 1 per transition) versus the UX benefit (zero — gaps are invisible).
- **No "release restores pre-claim position."** Release always lands at the bottom of Up-for-grabs. If users expect "undo-claim restores original slot," that's a separate design (needs a stored pre-claim slot per user or a snapshot mechanism).
- **No Snoozed reorder.** Snoozed stays sorted by `createdAt`.
- **No watchOS reorder.** iPhone only, like the first slice.
- **No change to the push/refresh-suspension plumbing.** The mid-drag suspension from PR #43 already works for any section.
