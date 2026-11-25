# Recurring chores — plan

## Context

Hearth's MVP explicitly deferred recurring chores. The design canvas ("Hearth refresh", artifact `E3m3Mfwr76tVnN75FPrpvR`) now picks a direction — **RecurRail** (`project/RecurRail.dc.html`, titled "Recurring A. Rail (chosen)") — and this slice brings it in. Two user-facing surfaces change:

- **Backlog** gains a `Recurring` section pinned above `Claimed` and `Up for grabs`: a horizontal, non-wrapping rail of circular icon tiles. Tapping a tile claims it for the tapper — the tile lifts and shrinks, the item appears in Claimed as a mint-tinted row with a small "Recurring" label beside the title. Completing that row folds it up and the tile flies back to the rail.
- **New chore** gains a `Recurring?` switch above the existing `I'll take it` switch. When Recurring is on, the add CTA produces a recurring chore (lands in the rail, nobody claimed, no will-complete-by). The `I'll take it` row and `By when?` reveal are hidden while Recurring is on.

Why now: the household is asking for everyday things ("make the bed", "unload the dishwasher") to stop being one-off list entries that pile up in Done. The rail makes them permanent fixtures that never leave the backlog.

**Settled product decisions (from the design conversation, 2026-11-23):**

1. **No cadence.** A recurring chore sits in the rail, period. It replenishes (returns to the rail) only when it is completed. There is no daily reset, no weekday schedule, no "done today" lingering badge persisted across a date boundary.
2. **Shared household.** The rail is the shared list's rail. Anyone in the list can claim any tile; when anyone completes it, it returns to the rail for everyone.
3. **Claim shape matches the design.** A claimed recurring chore uses the same Claimed row as a one-off, plus a small green "Recurring" label beneath the title. No will-complete-by chip. Completion unclaims and replenishes — no graveyard row, no entry in Done.
4. **Icon for MVP.** The design shows per-chore SVG icons (bed, dishwasher, plants, counters, trash, cat). MVP ships with a **single default SF Symbol** (`arrow.triangle.2.circlepath`) on every recurring tile. A per-chore icon picker is a follow-up slice; the schema carries `icon TEXT` from day one so it's additive. Deliberate deviation from the design — flagged so it doesn't drift back in by accident.
5. **Deviation from the design's "done today" state.** The design's `doneToday` badge and "back again tomorrow" toast assume a daily cycle. With no cadence, we drop both: on completion the tile animates back into the rail and the toast reads "**{title} · back in the rail**". The lift/return animations themselves stay.

### UI mocks (ASCII, iPhone 13 mini @ 390×844)

**Backlog with Recurring rail** (`RecurRail.dc.html`):

```
╭──────────────────────────────────────────╮
│                                          │
│  Saturday, October 24                    │
│  Backlog                                 │
│                                          │
│  ↻  Recurring    3 to go                 │
│  ╭──╮  ╭──╮  ╭──╮  ╭──╮  ╭──╮            │
│  │🛏 │  │🍽 │  │🪴│  │✨│  │🗑✓│  → scroll │
│  ╰──╯  ╰──╯  ╰──╯  ╰──╯  ╰──╯            │
│  Make   Unload Water  Wipe   Take        │
│  the    the    the    the    out         │
│  bed    dish…  plants count… trash       │
│                                          │
│  •  Claimed    4                         │
│  ╭──────────────────────────────────────╮│
│  │○  Feed the cat                   (M) ││
│  │   ↻ Recurring                        ││
│  ╰──────────────────────────────────────╯│
│  ╭──────────────────────────────────────╮│
│  │○  Clean out the fridge  [2d late](V) ││
│  ╰──────────────────────────────────────╯│
│  ╭──────────────────────────────────────╮│
│  │○  Mop the kitchen                (M) ││
│  ╰──────────────────────────────────────╯│
│                                          │
│  •  Up for grabs    2                    │
│  ╭──────────────────────────────────────╮│
│  │⌾  Wipe down the stove        [Claim]││
│  ╰──────────────────────────────────────╯│
│                                          │
│         ╭─────────────────╮         ╭──╮ │
│         │ ≡ Backlog · Done│         │ +│ │
│         ╰─────────────────╯         ╰──╯ │
╰──────────────────────────────────────────╯
```

- Tiles are 76pt wide, 56pt mint circle, teal icon (24pt), 2-line title clipped at 30pt.
- "3 to go" = count of unclaimed recurring tiles.
- A trash tile with a small green check overlay in the design is dropped for MVP (no done-today state — tile returns live immediately on complete).
- Claimed recurring row: mint background (`#EEF8F6`), no shadow, "↻ Recurring" label under the title (teal). Claimed one-off: white, drop shadow, optional peach "late" chip.
- `○` = tappable completion ring (claimant's view); `(M)`/`(V)` = assignee avatar.

**Tap → claim animation**: tile shrinks to 0.5× and translates down 34pt with fade over ~380ms, then the Claimed row springs in from above (translateY(-18), scale .95, 600ms spring).

**Complete → replenish animation**: row's ring fills green (22px dash draw-in, 280ms), row folds up (opacity 0 + max-height 0 + translateY(-24) scale .94, 450ms), then the tile reappears in its original rail slot with the design's `.back` keyframe (scale from 0.4 + translateY(30), 700ms spring). Toast: "Make the bed · back in the rail".

**New chore card with Recurring toggle** (`Create.dc.html`, morphed-open state):

```
╭──────────────────────────────────────────╮
│  New chore                           ╳   │
│                                          │
│  Descale the kettle                      │
│  ─────────────────────────────────────── │
│  Add a note, if it helps                 │
│                                          │
│  Recurring?                         ◯──● │  ← NEW
│                                          │
│  I'll take it                       ●──◯ │  ← hidden when Recurring=on
│  Claim it as soon as it's added          │
│                                          │
│  By when?                                │  ← hidden when Recurring=on
│  (Tonight)(Tomorrow)(This weekend)(No rush)│   or when I'll-take-it=off
│                                          │
│  ╭──────────────────────────────────────╮│
│  │          Add to backlog              ││
│  ╰──────────────────────────────────────╯│
╰──────────────────────────────────────────╯
```

Toggle heights, order from top: Recurring? → I'll take it → By when? reveal. Card height morphs:

| Recurring | I'll take it | Height |
|---|---|---|
| off | off  | 368pt |
| off | on   | 448pt |
| on  | —    | 288pt (I'll-take-it row + reveal both hidden) |

Toggle styling matches the existing one in `Create.dc.html`: lavender knob `#8F78E0` on, grey track `#DCD9E5` off, 48×28 track, 22×22 knob with 20pt travel.

**Submit copy**:
- Recurring on → toast "Added to Recurring".
- Recurring off, I'll take it on → "Added. Yours by {when}" (existing).
- Recurring off, I'll take it off → "Added. Up for grabs" (existing).

---

**Data-model choice.** One boolean column on `chores` — `is_recurring INTEGER NOT NULL DEFAULT 0`. A recurring chore is never marked completed in the `state` column: completing it clears `claimed_by`/`claimed_at`/`will_complete_by`/`section_order` and logs an event, leaving `state = 'open'`. A separate `recurring_chores` table was considered and rejected — doubles every join and buys nothing over a boolean. `icon TEXT` lands alongside for the future picker.

---

## Slice — Recurring chores

**User-facing value**: on the New chore card, flip "Recurring?" and add. The chore appears as a circular tile in a new `Recurring` row at the top of the Backlog. Any household member can tap the tile to claim it — the tile lifts away and the chore appears as a mint-tinted row under Claimed with a small "Recurring" label. Checking the ring completes it; the row folds up, the tile flies back into the rail with a "back in the rail" toast, and every other member's device refetches within a couple of seconds.

**Branch base**: `main`, fresh pull. Before branching: `git checkout main && git pull --ff-only origin main`; record the resulting HEAD SHA in the PR description.
**Branch**: `recurring-chores`

One slice, one PR. Server + iOS ship together so the slice is user-facing on merge. TDD on the server side. iOS is view-level work verified on an iPhone 13 mini simulator per the Build Plan's design-review standard.

### Checklist — Server (`server/...`)

- [ ] **Migration** `server/migrations/00008_add_chores_recurring.sql`
  - [ ] `ALTER TABLE chores ADD COLUMN is_recurring INTEGER NOT NULL DEFAULT 0 CHECK (is_recurring IN (0,1))`
  - [ ] `ALTER TABLE chores ADD COLUMN icon TEXT` (nullable; default SF Symbol applied client-side when NULL)
  - [ ] `CREATE INDEX chores_list_recurring_open ON chores(list_id) WHERE is_recurring = 1 AND state = 'open' AND claimed_at IS NULL` — hot path for the rail query
  - [ ] Backfill is a no-op (existing rows stay `is_recurring = 0`)
- [ ] **Events** — in the same migration, extend `events.type` CHECK to add `'replenished'`. Fires when a recurring chore returns to the rail after completion. Keep `'completed'` firing as well for audit continuity.
- [ ] **Queries** in `server/queries/chores.sql`
  - [ ] Add `is_recurring, icon` to every SELECT column list that returns a chore (`CreateChore`, `BacklogForList`, `BacklogForUser`, `GetChoreByID`, `ClaimChore`, `ReleaseClaim`, `SetWillCompleteBy`, `CompleteChore`, `AssignNextSectionOrder`, …). Mechanical, one grep.
  - [ ] `CreateRecurringChore` — `INSERT INTO chores (id, list_id, creator_id, title, notes, is_recurring, icon) VALUES (?, ?, ?, ?, ?, 1, ?)`. Keeps `CreateChore` a one-off-only path so the handler can stay simple.
  - [ ] `ReplenishRecurring` — single statement clearing `claimed_by`, `claimed_at`, `will_complete_by`, `section_order`, guarded by `WHERE id = ? AND is_recurring = 1 AND state = 'open'`. Returns the row.
- [ ] **Handler wiring** in `server/internal/api/chores.go`
  - [ ] `handleCreateChore` — accept `is_recurring: bool` and optional `icon: string?`. If `is_recurring = true`, route to `CreateRecurringChore`; reject `claimed_by` / `will_complete_by` on the same request as 400 ("a recurring chore cannot be claimed at creation"). Dispatch the usual `created` event + silent push.
  - [ ] `handleCompleteChore` — if the row has `is_recurring = 1`, run `ReplenishRecurring` **instead of** the state=completed UPDATE, and emit a `replenished` event in the same transaction. The chore stays `state = 'open'`. Silent push dispatches as today. The visible completion push still fires with existing copy — household members still want the "Priya completed 'Water the plants'" alert even though the chore lives on.
  - [ ] `handleClaimChore` — unchanged path; a recurring row claims the same way as a one-off (`AssignNextSectionOrder` still runs so the row lands at the bottom of Claimed).
  - [ ] DTO (`toChoreDTO`) exposes `is_recurring` (snake_case) and `icon`.
- [ ] **Tests** (TDD — write first) in `server/internal/api/chores_recurring_test.go`
  - [ ] Create recurring: POST `/chores` with `{title, is_recurring: true}` → 200, row has `is_recurring=1`, `claimed_by IS NULL`, `state='open'`.
  - [ ] Reject bad create: `{is_recurring: true, claimed_by: <uid>}` → 400.
  - [ ] Backlog lists a recurring, unclaimed chore and flags `is_recurring=true` on the DTO.
  - [ ] Claim → complete flow: member A creates recurring; member B claims (section_order assigned); member B completes → row is still `state='open'`, `claimed_by IS NULL`, `claimed_at IS NULL`, `section_order IS NULL`; a `replenished` event exists.
  - [ ] Visible push alert fires on complete even though the chore replenished (existing push fake records the alert).
  - [ ] `GET /chores` by member C (the actor-excluded silent-push target) sees the replenished tile in the rail.
  - [ ] Non-recurring chore complete still goes to graveyard (regression guard on the existing path).

### Checklist — iOS (`apps/HearthApp/...`)

- [ ] **Model** — `Models.swift`
  - [ ] Add `isRecurring: Bool` (decoded from `is_recurring`) and `icon: String?` to `HearthChore`. Default to `false` / `nil` on decode for forward-compat with older server responses.
  - [ ] Thread through `CodingKeys` and preview fixtures.
- [ ] **Backlog sections** — `BacklogSections.swift`
  - [ ] New `recurring` slot: chores where `isRecurring && claimedBy == nil`. Excluded from both `claimed` and `upForGrabs`.
  - [ ] A recurring-and-claimed chore belongs in `claimed`, shown with the new "Recurring" label.
  - [ ] Order within the rail: `createdAt ASC` (oldest first). No `section_order` for rail entries — reorder is a Claimed feature.
- [ ] **Backlog view** — `BacklogView.swift`
  - [ ] Above the existing Claimed section, render a `RecurringRailView` when `recurring` is non-empty. Hidden otherwise.
  - [ ] Section header: `arrow.triangle.2.circlepath` SF Symbol (teal `#1A5E54`), label "Recurring", trailing muted count ("3 to go").
- [ ] **New file** — `RecurringRailView.swift`
  - [ ] Horizontal `ScrollView(.horizontal, showsIndicators: false)` of `RecurringTile` buttons, 76pt wide each, 4pt gap, 14pt leading/trailing inset.
  - [ ] Tile: 56pt circle, mint fill (`#CDEEE7` light / palette-dark equivalent), 24pt SF Symbol centered (teal `#1A5E54`). Below: title, 12pt Geist medium, 2 lines, 30pt height clip, center-aligned. Fixed-height matches design.
  - [ ] Tap handler: `.snappy` scale-to-0.5 + translateY(34) + opacity 0 over ~380ms (the design's `.lift` keyframe), then `store.claim(chore)`.
  - [ ] On a newly-replenished tile (incoming after a complete), apply the `.back` keyframe: scale-from-0.4 + translateY(30) → identity, 700ms spring. Driven by an id-set of "recently replenished" chores in the store.
  - [ ] Accessibility: each tile is a `Button` with label "Claim {title}"; icon `.accessibilityHidden(true)`.
- [ ] **New file** — `RecurringTile.swift`
  - [ ] Pure view, no state. Takes `title`, `iconSymbol`, `onTap`, `state: .idle | .leaving | .arriving`.
  - [ ] Default `iconSymbol` is `"arrow.triangle.2.circlepath"` when `chore.icon == nil` (MVP default).
- [ ] **Chore row** — `ChoreRow.swift` / `ChoreRowModel.swift`
  - [ ] When `chore.isRecurring && chore.claimedBy != nil`, render under the title a horizontal `HStack`: `arrow.triangle.2.circlepath` 11pt + "Recurring", Geist 12pt semibold, teal `#1A5E54`.
  - [ ] Recurring claimed row uses mint background (`#EEF8F6`) with `box-shadow: none`, matching the design's distinction from one-off rows.
  - [ ] Recurring claimed row **omits** the overdue peach chip (no will-complete-by to be overdue against).
- [ ] **New chore card** — `NewChoreCard.swift`
  - [ ] Add a `Recurring?` `Toggle` row above the `I'll take it` row. Same lavender knob (`#8F78E0` on, `#DCD9E5` off).
  - [ ] When Recurring is on: hide the entire `I'll take it` row **and** the `By when?` reveal (`isHidden` + `withAnimation`, matching the design's reveal animation).
  - [ ] Card height reflows — the design's `mh: s.mine ? 448 : 368` becomes `recurring ? 288 : (mine ? 448 : 368)`. Measure in the simulator.
  - [ ] On submit:
    - [ ] If `recurring`: POST `/chores` with `is_recurring: true, icon: nil`. Toast: "Added to Recurring".
    - [ ] Else: current flow (unchanged).
  - [ ] Clear the toggle on close so the next open starts with Recurring off.
- [ ] **Store** — `BacklogStore.swift`
  - [ ] `createChore(…, isRecurring: Bool)` — adds the flag to the request body.
  - [ ] `complete(chore)` — unchanged client-side; the server decides replenish vs graveyard. On the response, if the returned row has `claimed_by == nil && is_recurring`, inject the id into a `recentlyReplenished` set (cleared on next refetch or after 1200ms) so the Rail can trigger the `.back` animation.
- [ ] **Done view** — `GraveyardView.swift` / `GraveyardSections.swift`
  - [ ] No change expected. A replenished recurring chore does not land in the graveyard. Verify `GraveyardStore` filters on `state = 'completed'` server-side and does not surface replenish events as rows.
- [ ] **Design review** per Build Plan standard
  - [ ] iPhone 13 mini, light and dark, screenshots of Backlog with ≥3 recurring tiles + 1 recurring claimed + 1 one-off claimed
  - [ ] Capture tap → claim animation and complete → return-to-rail animation
  - [ ] Haptics: `.light` on tile tap, `.success` on complete (same as today's claim/complete)

### Watch (`apps/HearthWatch/...`)

Deferred. The Watch's "my chores" scope renders claimed items — a recurring chore claimed by the wearer shows up there by its claimed state alone, so Watch inherits recurring chores "for free" without a rail of its own. A proper Watch rail is a follow-up if the household uses the feature heavily.

**Verification on Watch during this slice**: claim a recurring chore on iOS, confirm it appears in the Watch's list; complete from Watch, confirm iOS Backlog updates and the tile returns to the rail.

### Acceptance

- [ ] Server tests green on `just test` with fresh `./ci.db`.
- [ ] iOS builds green for `HearthApp` and `HearthWatch`.
- [ ] Manual on iPhone 13 mini sim:
  - [ ] Create a recurring chore → appears as a tile in the rail with default icon.
  - [ ] Claim it. Tile lifts + shrinks; a mint "Recurring" row appears in Claimed.
  - [ ] Complete it. Row folds; tile flies back; toast reads "{title} · back in the rail".
  - [ ] From a second test account on the same list, confirm the sequence propagates (silent push triggers refetch).
  - [ ] Create a non-recurring chore and complete it; confirm it still lands in Done (no regression).
- [ ] TestFlight build auto-uploads on merge to main.

### Risks & open questions

- **Icon picker.** Shipping with one default icon is intentional but noticeable — the design implies diversity. If first-use feedback is "every tile looks the same," the picker becomes the very next slice. Schema is ready.
- **Rail ordering.** Currently `createdAt ASC`. If the rail grows past ~10 tiles, we may want drag-reorder on the rail itself; reuses `section_order` trivially since `ReplenishRecurring` already clears it.
- **Completion from Watch.** Existing complete endpoint covers it; only wrinkle is the Watch's local optimistic strike-through, which assumes removal from "my list." For a recurring chore it should animate the strike-through, then remove — visually the same as today. Flagged for a 2-min check in the device pass.
- **Audit history.** Completing a recurring chore emits a `replenished` event in addition to (not instead of) the existing `completed` event. If downstream analytics count completions, both fire.
- **Deletion of a claimed recurring chore.** Out of scope; existing creator-delete rules apply. If the creator deletes while someone else has it claimed, the row goes away for everyone, same as today for a one-off.
