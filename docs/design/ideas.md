# Hearth — UI/UX design ideas to revisit

A living log of design directions we've discussed but not implemented. **Nothing here is on the roadmap** — the current build plan (`docs/plans/BUILD_PLAN.md`) is the committed scope. Revisit after Slice 10 (or earlier if a slice obviously benefits).

Design thesis across all of these: the UI should be **fun and delightful** — bold and new, but fitting the user's actual flow like a glove. Not gimmicky, not Material-Design-scrubbed-Apple-style default; something that makes using Hearth feel like a tiny ritual.

---

## Direct-expansion patterns: UI *becomes* the next UI, not pushes it

The seed idea: tapping the FAB makes the button itself grow and morph into the create-chore form, instead of presenting a modal sheet. Generalizes to:

- **Chore row → detail**: tap a row and it expands in place (unfolds like a drawer) to show metadata and actions. No modal, no push. The row *is* the detail view.
- **Sign-out button → confirmation**: tap the sign-out icon and a small confirmation widget grows out from it, instead of a system alert interrupting flow.
- **Completed chore → graveyard entry**: when you complete a chore, the row doesn't vanish instantly — it shrinks, fades, and slides into the "Done" tab icon with a trail. The user sees exactly where it went.

**Why**: respects the user's spatial memory. UI changes feel like physical consequences of their action, not jump-cuts.

**Cost**: matchedGeometryEffect + custom transitions throughout. Not trivial but SwiftUI-native.

---

## Alternative chore layouts (non-list)

The seed critique: a vertical scrolling list is the default and it's boring. These are alternatives — but keep the list; make these additional views you can toggle to.

### Card deck

One chore at a time, big and tangible. Swipe right to claim, up to complete, down to snooze. Tinder-for-chores. Great for "I only care about what's next" moments.

**Caveat (from user)**: doesn't let you see all chores at once. The boring list wins on scannability. So — card deck is a *mode*, not a replacement.

**When to pick deck mode**: morning tap-to-start-day, or "give me the next thing" during a chore-doing session.

### Floor plan

Chores pinned to rooms of the user's house (dishes/trash → kitchen; dust → living room; tiles → bathroom). Tap a room, see its chores. Mirrors how people actually spatially organize housework.

**Caveat (from user)**: getting a floor plan INTO the app is tedious and maybe impossible. Options we'd need to figure out:
- User sketches rooms with their finger (lofi, charming, imperfect)
- Pre-set room templates ("1BR apartment," "3BR house") the user picks from and labels
- Just a conceptual plan — not a to-scale layout, but labeled bubbles for rooms the user types in

**Verdict**: cool metaphor, hard to deliver without one of the three concessions above. Deferred.

### Eisenhower matrix (user said: practical but hard to visualize)

The classical Eisenhower matrix is Urgent × Important, which doesn't map directly to chores (we don't store "importance"). Reframed for chores, the axes become:

- **X axis**: Due soon ← → Later / undated
- **Y axis**: Unclaimed ← → Claimed

Four quadrants:

```
                 Unclaimed
                     |
   Overdue +         |     Later +
   unclaimed         |     unclaimed
   (ohshit)          |     (backlog)
                     |
   -----Due soon-----+----Later-----
                     |
   Overdue +         |     Later +
   claimed           |     claimed
   (in flight)       |     (committed)
                     |
                  Claimed
```

Chores appear as colored dots in their quadrant, sized by how overdue they are (or how old). Top-left glows red (the "we need to deal with this NOW" zone). Bottom-right is calm green ("under control"). The quadrant you need to look at is visually obvious at a glance.

**Alt visualization — stream**: instead of 4 cells, show a horizontal stream flowing left-to-right. Chores at the right-edge are urgent-unclaimed (about to fall off the cliff). Chores at the left-edge are later-committed (chill). As time passes, chores drift rightward. More ambient, less grid-y.

**Verdict**: worth prototyping once we have claim + will_complete_by (Slice 5–6). Quadrant version is simpler to build; stream version is more delightful.

---

## Delightful micro-interactions

### Pinch-to-expand

Pinch out on a chore row → it zooms into a full-screen detail. Pinch in → zoom back out. Direct manipulation, physical. Replaces push-navigation for drill-down.

**Pairs well with**: the row-expands-in-place pattern above. Pinch is the "give me more detail" gesture even when you're not on a row.

### Pull-down pulls

Instead of a spinner on pull-to-refresh, the chores physically stretch downward like you're pulling a sheet of fresh ones from a drawer. The elastic drag gives feedback that *you* are the refresh mechanism. No circle, no icon — just the content behaving.

**When**: on BacklogView whenever we have real network fetches worth animating.

---

## Deferred (interesting but less immediately promising)

- **Jar of marbles** — chores as marbles in a jar you can shake. Playful but low scannability.
- **Garden metaphor** — each chore is a plant that wilts if overdue. Charming but requires an art pass.
- **Campfire** — household members sit around a fire, chores are logs. Beautiful social metaphor but hard to make legible.
- **Relay baton** — chores physically pass between people on claim/assign. Compelling but we dropped assignment from MVP, so half the metaphor evaporates.

---

## Principles I keep landing on

- **Boring plumbing, delightful meat.** Sign-in / errors / loading should be austere and fast. Create / claim / complete should surprise and please.
- **Record facts, derive delight.** The data model stays simple; the visual flourishes are rendering choices on top of plain fields.
- **Taste over gimmick.** Everything here should fit the Hearth voice (warm, calm, home-y). No confetti, no streak badges, no leaderboard. Micro-interactions, not maximalist celebration.
- **One alternative at a time.** Don't pile up card-deck + floor-plan + matrix as competing views; introduce one at a time and see whether it earns its place.

---

## Next review

After Slice 5 (claim + complete) and Slice 6 (will_complete_by). Those unlock the state richness that most of these ideas depend on (semantic colors, overdue, quadrant zones, state-change animations).
