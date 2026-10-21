# Hearth — Build Plan

**Project name**: Hearth. Repo, Go module, iOS bundle, and Watch bundle all share the `hearth` root. Go module path: `github.com/<org>/hearth/server`. iOS bundle: `com.<org>.hearth` (watchOS: `com.<org>.hearth.watchkitapp`).


## Context

Build an iOS 26+ app (iPhone 13 mini and above) with an Apple Watch companion for shared household chore tracking. Multiple members share one list, create/claim/assign/complete items, snooze them personally (8h fixed), and get push notifications on activity. Backend is Go + SQLite on a single Digital Ocean droplet, deployed natively (binary + systemd) and reverse-proxied by the droplet's existing Caddy. The outcome is a polished v1 deployed to a real household.

The work is structured as **small, complete, user-facing slices**. Each slice is a branch → PR → merge → auto-deploy → verify-on-device cycle. TDD throughout: write the first failing test, make it pass, repeat. Database tests run against a real SQLite file with per-test transactions (no mocks). External services (APNs, Apple's JWKs endpoint) are mocked at the Go interface boundary.

**Database choice**: SQLite via `modernc.org/sqlite` (pure Go, no CGo), chosen over Postgres for simplicity and $0 infra cost at household scale (~1 household, 2–6 users). See [DATA_MODEL.md](../DATA_MODEL.md) for details. The migration path to Postgres is kept open if "multi-household" ever becomes a goal.

## Repo structure (to be created)

```
hearth/
├── .git/                             # initialized in Slice 0, first task
├── docs/{DATA_MODEL.md, SLICES.md}
├── server/
│   ├── cmd/hearth/main.go
│   ├── internal/{api,auth,db,items,users,notifications}/
│   ├── queries/*.sql
│   ├── migrations/*.sql
│   ├── sqlc.yaml, go.mod, Dockerfile
├── apps/Hearth.xcworkspace/{HearthApp, HearthWatch, HearthAppTests}
├── deploy/{Caddyfile.hearth, hearth.service, droplet/}
├── .github/workflows/
├── .gitignore
└── justfile
```

## Visual design and interaction (cross-cutting standard)

Design is not a Slice 10 concern — every UI-touching slice meets this bar from the first commit. Slice 10 is a final audit pass, not where polish starts.

**Aesthetic**: sleek, modern, minimal. No gratuitous chrome. Content first, controls recede until needed.

**Material**: Liquid Glass (iOS 26). Nav bars, sheets, floating action elements, and tab bars use the system Liquid Glass material (`.glassEffect()` / `.backgroundExtensionEffect()` as appropriate). Content behind glass is designed to tint pleasantly — chosen colors are muted enough to stay clean through the material.

**Palette** (proposed, confirm when first used in Slice 3):

| Role | Light | Dark |
|---|---|---|
| Background (warm, not pure) | `#FBF8F3` | `#1C1A17` |
| Lavender — primary / "mine" | `#C9B6E4` | `#8A7BA8` |
| Mint — success / completed | `#B6E4CF` | `#7AA896` |
| Peach — warning / overdue | `#F4C4A3` | `#B88A6E` |
| Sky — info / claimed by other | `#A3C9F4` | `#6B94C2` |
| Text primary | `#2A2621` | `#F0ECE6` |
| Text secondary | `#6B6560` | `#A6A09A` |

Colors are semantic, never decorative. A lavender highlight **always** means "this is yours." A peach badge **always** means "overdue." Users learn the system by using it.

**Typography**: SF Pro, system defaults for Dynamic Type. Titles slightly heavier than default (`.rounded` design where it reads warm). Minimum 17pt for primary content.

**Animation and transitions**: every state change is animated, not snapped. Specifics:
- Spring animations (`.snappy` for taps, `.bouncy` for sheet presentation, matched-geometry for item moves between backlog ↔ personal ↔ graveyard).
- Haptic feedback on every meaningful action: `.light` on claim, `.success` on complete, `.soft` on snooze.
- Swipe gestures are consistent across screens:
  - **Swipe right → claim** (lavender hand symbol). On the backlog, swiping right on an unclaimed item claims it.
  - **Swipe left → snooze** (sky moon symbol). On the personal list, swiping left snoozes an item for 8h. Not available on the backlog.
  - **Complete** (mint check symbol) is a tap on a "mark done" affordance in the row, not a swipe — completion is a deliberate action, not a dismiss gesture.
- No instant deletions. Items fade and collapse (~250ms) when leaving a list.
- Pull-to-refresh uses a custom pastel-tinted indicator, not the system default gray.

**Watch**: smaller surface, same material and palette. Prefer SF Symbols at scale over text. Haptics on every action — the Watch's primary feedback channel.

**Design review per slice**: a slice that touches UI isn't done until (a) it looks right on an iPhone 13 mini (smallest supported screen) in both light and dark mode, (b) interactions have animation and haptics, (c) the palette is applied semantically. Record a short screen capture demonstrating all three in the PR description.

## Locked decisions

- **Session tokens**: server-signed JWT, 30-day expiry, no refresh. User re-signs-in monthly. Revisit if abuse observed.
- **Invite flow**: 6-digit short-lived code + universal link. No QR for MVP.
- **Will-complete-by on unclaimed item**: implicit claim (matches user intent, one tap instead of two).
- **Notification dispatch**: synchronous within request handler, in a `go func()` with `recover()`, fire-and-forget. Not blocking HTTP response. Revisit if event volume grows.
- **Silent cache-invalidation push payload**: `{list_id, last_event_id}` — client fetches events since the given ID.
- **APNs test boundary**: `Pusher` interface in `internal/notifications`. Unit tests use a fake that records calls. Sandbox integration tests behind a build tag (`-tags=apns_live`), run manually before release, not in CI.
- **Timestamp handling**: UTC in SQLite as ISO-8601 `TEXT`, display local on client.
- **Delete semantics**: creator-delete is hard delete; complete is soft (graveyard).
- **Memberships for MVP**: all members equal, no owner / no kick. Add later.

## Pre-work (before Slice 0)

1. **Create the repo**: `mkdir hearth && cd hearth && git init`, add `.gitignore` (Go + Swift + Xcode user data + `.env` + OS cruft) and a short `README.md`. Create the GitHub repo (private initially), push the initial commit on `main`.
2. **Write `docs/DATA_MODEL.md`**: entities, relationships, indexes, and invariants — Users, Lists, ListMemberships (future-proofed many-to-many), Items (no `due_date`, only `will_complete_by`), Snoozes (`PK (user_id, item_id)`, upsert), Events (append-only), DeviceTokens (per-user per-platform). Call out the FTS5 virtual table on `items.title` for dedup search, the partial index on `item (list_id) WHERE state = 'open'`, and the `snooze (user_id, snoozed_until)` index. This document is the shared reference for all slice work. Commit on `main`.

## Slice 0 — Foundation (prerequisite, NOT user-facing)

Keep ruthlessly small. Only what Slice 1 needs.

- **Git**: initialize the repo at `hearth/`, create `.gitignore` (Go, Swift, Xcode user data, `.env`, `node_modules` just in case, OS cruft), add a short `README.md`, create a GitHub repo and push the initial commit. Default branch `main`. Branch per slice (`slice-0-foundation`, `slice-1-deploy`, ...).
- Backend: `go.mod` (`github.com/<org>/hearth/server`), chi skeleton, SQLite via `modernc.org/sqlite` (pure Go, no CGo), stdlib `slog`, `sqlc.yaml` wired for SQLite, migrations runner (`goose`), `/healthz` endpoint.
- iOS: `Hearth.xcworkspace` with `HearthApp` (iOS 26+) and `HearthWatch` (watchOS) targets, SwiftUI App skeleton, base Codable types, URLSession client stub with bearer-token placeholder, Swift Testing target (`HearthAppTests`).
- Infra: `justfile` with `just test`, `just migrate`, `just run`, `just dev`, `just db-reset`. Local DB is just a file at `./hearth.dev.db` — no daemon.
- CI: GitHub Actions runs `go test ./...` against a fresh SQLite file (no service container needed), and `xcodebuild build` for iOS and watchOS targets.

**First failing test**: a Go test that opens a transaction via the shared test harness, inserts a row into a temp table, rolls back on completion. This is the harness every slice after depends on.

**Acceptance**: `just test` is green on a fresh clone. `/healthz` returns 200 locally.

## Slice 1 — Deploy "Hello Chores" to the droplet

**User-facing value**: anyone can hit `https://<domain>/healthz` from a phone browser and see it works.

- Infra: provision the existing droplet (install Go-built binary dir at `/opt/hearth`, `deploy` user with narrow sudoers, systemd unit, Caddy site file at `/etc/caddy/sites/hearth.conf` reverse-proxying `api.hearth.prakashvenkat.com` to `localhost:8080`). SQLite file at `/opt/hearth/data/hearth.db`. Deploy workflow (GitHub Actions SSH + scp) runs on every merge to main. Secrets live in GitHub Actions.
- Backend: nothing new beyond `/healthz`.
- iOS: nothing yet.

**First failing test**: CI post-deploy smoke test — `curl https://<domain>/healthz` must return 200 or the deploy workflow fails.

**Acceptance**: push to main → service is live within ~2 min, TLS valid, migrations ran on startup.

## Slice 2 — Sign in with Apple end-to-end

**User-facing value**: open the app, tap "Sign in with Apple," land on a signed-in screen.

- Backend: `users` migration (`id`, `apple_sub` unique, `display_name`, timestamps). Apple JWT verifier: fetch Apple JWKs with caching, verify signature + `aud` + `iss` + `exp` + clock skew tolerance. `POST /auth/apple` is idempotent by `apple_sub`, returns a server-signed session JWT (30-day). `GET /me` behind auth middleware.
- iOS: `AuthenticationServices` button, Keychain-stored session token, auth-gated root view.

**First failing tests (backend, in order)**:
1. JWT verifier accepts a fixture JWT signed by a test key when JWKs endpoint is swapped via DI.
2. JWT verifier rejects: expired, wrong-aud, wrong-iss, bad-signature.
3. `POST /auth/apple` creates user on first call; returns the same user on second call with same `apple_sub`.
4. Auth middleware rejects missing / expired / wrong-signature session tokens.

**Acceptance**: fresh install → sign in → `/me` returns user → relaunch keeps session.

## Slice 3 — Create and view items (single user)

**User-facing value**: add "Take out trash," see it in the backlog, persists across relaunches.

- Backend: `lists`, `list_memberships`, `items`, `events` migrations. First sign-in lazily auto-provisions a personal list and membership. `POST /items`, `GET /items` returning the backlog (all non-completed items in the user's list). Event rows written in the **same transaction** as the item mutation.
- iOS: Backlog screen (SwiftUI List with Liquid Glass background), "Add item" sheet presented with a bouncy spring, Codable `Item` type, repository layer. Palette applied here for the first time — background `#FBF8F3` / `#1C1A17`, item rows plain with secondary-text metadata, FAB uses Liquid Glass. Light haptic on tap-to-add. Confirm palette with user when this slice is PR-ready.

**First failing tests**:
- Backend: creating an item without an existing list auto-provisions one; `GET /items` returns only items in the caller's list; create-item + event-row are atomic (test rollback drops both).
- iOS: repository happy path with stubbed HTTP; loading / error / empty states render correctly.

**Acceptance**: create 3 items, kill app, relaunch, items still there. Deployed to droplet, works end-to-end from the phone.

## Slice 4 — Multi-user: invite flow + shared list

**User-facing value**: invite a partner with a 6-digit code or link, they join, both see each other's items.

- Backend: `POST /invites` returns a short-lived (24h) 6-digit code tied to the caller's list. `POST /invites/:code/accept` adds a membership (idempotent). `GET /items` now scopes by list membership rather than ownership. Universal link setup: `apple-app-site-association` served by Caddy, team ID + bundle ID + associated-domains entitlement aligned.
- iOS: Settings → "Invite someone" screen showing the code + share-sheet button. Deep-link handler on launch that extracts a code and offers to accept.

**First failing tests**:
- Expired code returns 410; nonexistent code returns 404; double-accept is a no-op (same user accepting twice doesn't duplicate membership).
- Backlog scoping: Alice's items appear in Bob's backlog only after Bob accepts her code.

**Acceptance**: two devices, two Apple IDs, one shared list.

## Slice 5 — Claim and complete

**User-facing value**: Bob sees Alice's item, taps "I'll do it," taps complete when done. It lands in the graveyard. Alice finds it when searching.

- Backend: `PATCH /items/:id/claim` (conditional update `WHERE claimed_by IS NULL` to prevent races), `POST /items/:id/complete`, `GET /items/graveyard?q=` using the FTS5 virtual table for substring search. Events for claim and complete.
- iOS: **swipe right → claim** (lavender hand) on the backlog. Complete is a tap on a row-level affordance (mint check), not a swipe. Graveyard tab with search bar. Visual states: unclaimed is plain, claimed-by-me gets a lavender accent, claimed-by-someone gets a sky-tinted avatar chip. Items animate between lists via `.matchedGeometryEffect` — claiming slides the row into the personal view, completing fades + collapses then surfaces in graveyard. Light haptic on claim, success haptic on complete.

**First failing tests**:
- Concurrent claim: two simultaneous `PATCH /claim` requests on the same unclaimed item — exactly one succeeds, the other returns 409.
- Complete removes the item from `GET /items` and makes it appear in `GET /items/graveyard`.
- Graveyard search is case-insensitive and matches substrings via trigram.

**Acceptance**: full solo lifecycle (create → claim → complete → find in graveyard) works. Two-user claim race test passes.

## Slice 6 — Assign + will-complete-by

**User-facing value**: Alice assigns an item to Bob. Bob commits "I'll have this done by Friday 6pm."

- Backend: `PATCH /items/:id/assign` (assignee must be a list member), `PATCH /items/:id/will-complete-by`. Events for both. Setting `will_complete_by` on an unclaimed item implicitly claims it for the caller. Overdue is derived (`will_complete_by < now() AND state = 'open'`), returned as a boolean on the item payload.
- iOS: assign picker (list members shown as pastel-tinted avatars), date picker for `will_complete_by` presented as a glass sheet. Overdue items get the peach treatment — a subtle peach left-border and peach-tinted "overdue" badge. Avatar of the assignee animates onto the row when assignment is made.

**First failing tests**:
- Assigning to a non-member returns 400.
- `PATCH /will-complete-by` on an unclaimed item claims it in the same transaction (both state changes atomic).
- Overdue computed correctly: item with `will_complete_by` yesterday is `overdue=true` for all list members.

**Acceptance**: Alice assigns, Bob sees it on his personal list, commits to a date, Alice sees Bob's commitment on her backlog view.

## Slice 7 — Snooze

**User-facing value**: swipe snooze on a chore. Not now. 8 hours later it's back.

- Backend: `POST /items/:id/snooze` upserts into `snoozes (user_id, item_id, snoozed_until)` with `snoozed_until = now() + 8h`. `GET /items/mine` (personal list) filters out items where the current user has a non-expired snooze. Event logged.
- iOS: **swipe left → snooze** on personal list only (not backlog), sky-tinted moon symbol. Snoozed items fade + collapse out of "my chores" (~250ms) with a soft haptic, and remain in backlog with a subtle sky "zzz" indicator. A snoozed item re-emerging after 8h animates back in with a bouncy spring.

**First failing tests**:
- Upsert: second snooze on the same item by the same user replaces the first's `snoozed_until` (one row, not two).
- Expired snoozes don't filter (an item snoozed 10h ago reappears).
- Snooze is per-user: Alice snoozing doesn't affect Bob's personal list.

**Acceptance**: snooze at 10am, gone from mine until ~6pm, then back.

## Slice 8 — Notifications (APNs)

**User-facing value**: Alice adds an item → Bob's phone buzzes.

- Backend: `device_tokens` table (`user_id`, `token`, `platform` enum of `ios`/`watchos`), `POST /devices` register endpoint, `Pusher` interface in `internal/notifications`, production impl using `sideshow/apns2`. Dispatcher consumes the `events` table (synchronous `go func` after event commit, with `recover()`). Visible push + silent `content-available: 1` push with `{list_id, last_event_id}` payload. Global on/off setting on `users`. Handle APNs 410 (Gone) → delete the token row.
- iOS: notification permission prompt on first sign-in, device token registration, foreground + background handlers, settings toggle for the global on/off.

**First failing tests**:
- Event write triggers a `Pusher.Push` call with the correct token set (all list members minus the actor).
- Users with `notifications_enabled=false` are skipped.
- APNs 410 response removes the token row.
- Multiple devices per user all get pushed.

**APNs testing strategy**:
- CI: fake `Pusher` only.
- Pre-release: `go test -tags=apns_live ./internal/notifications/...` hits APNs sandbox manually.
- Device check: send one of each notification type to real paired devices before each release.

**Acceptance**: end-to-end on two real devices — create → buzz. Toggle off → no buzz.

## Slice 9 — Apple Watch app

**User-facing value**: glance at wrist, see "my chores," tap to complete. Notification actions work.

- WatchOS target in the workspace. Independent URLSession client (not WatchConnectivity), bearer token synced via Keychain sharing with the iOS app.
- Views: "My chores" list (same query as iOS personal list), tap-to-complete. Same palette as iOS, condensed. Liquid Glass applied to the Watch's native list chrome. Haptics on every action (success on complete, soft on snooze).
- Notification categories with action buttons (Complete, Snooze), each with its semantic color on the action button.

**First failing tests**:
- Notification action handler for "Complete" calls `POST /items/:id/complete` with the correct item ID.
- "My chores" view model handles happy, empty, and error states.

**Acceptance**: buzz on wrist, swipe to complete, item is gone on iOS too.

## Slice 10 — Final audit + production hardening

Design polish is **not** introduced here — it's been baked in since Slice 3. This slice is a final sweep for consistency and the operational concerns that don't belong in any single earlier slice.

- Design audit: side-by-side screenshot comparison across every screen in light + dark on iPhone 13 mini, iPhone 15 Pro, and Watch. Flag any inconsistency in palette application, spacing, animation timing, or haptic coverage. Fix in this slice.
- Accessibility audit: VoiceOver pass on every screen, Dynamic Type at XXL, reduced-motion respected (animations degrade to crossfades, not disabled entirely).
- Structured logs with request IDs end-to-end, basic metrics endpoint.
- Rate limiting on `/auth/apple` and `/invites/*/accept`.
- Nightly SQLite backup (`sqlite3 hearth.db ".backup hearth.bak.db"` is safe under concurrent writes). Rsync the backup off-droplet to free storage (Backblaze B2 free tier, or the developer's local machine via a reverse-pull cron).
- iOS error reporting (OSLog + lightweight server-side error ingestion).
- TestFlight distribution set up.

## Critical files to be created (initial scaffold)

Representative paths — complete list grows per slice:

- `/Users/pvenkat/Projects/hearth/.gitignore` — pre-work (git init), before anything else.
- `/Users/pvenkat/Projects/hearth/README.md` — pre-work.
- `/Users/pvenkat/Projects/hearth/docs/DATA_MODEL.md` — pre-work, before Slice 0 backend/iOS tasks.
- `/Users/pvenkat/Projects/hearth/server/cmd/hearth/main.go` — Slice 0.
- `/Users/pvenkat/Projects/hearth/server/sqlc.yaml` — Slice 0.
- `/Users/pvenkat/Projects/hearth/server/internal/db/testdb.go` — the per-test-transaction harness; everything else depends on it. Slice 0.
- `/Users/pvenkat/Projects/hearth/server/migrations/0001_init.sql` — Slice 0 (empty-ish) and extended slice-by-slice.
- `/Users/pvenkat/Projects/hearth/deploy/hearth.service` — Slice 1 (systemd unit for the native binary).
- `/Users/pvenkat/Projects/hearth/deploy/Caddyfile` — Slice 1.
- `/Users/pvenkat/Projects/hearth/.github/workflows/deploy.yml` — Slice 1.
- `/Users/pvenkat/Projects/hearth/justfile` — Slice 0.
- `/Users/pvenkat/Projects/hearth/apps/Hearth.xcworkspace` — Slice 0 (empty), fleshed out slice-by-slice.

## Verification

Each slice ships green on these three gates before merge:

1. **Unit + integration tests pass**. `just test` is green: `go test ./...` against a fresh SQLite file, and `xcodebuild build` for iOS and watchOS. No skipped tests.
2. **Deployed smoke test passes**. GitHub Actions deploys to the droplet; a post-deploy curl against the relevant endpoint returns the expected status.
3. **Manual acceptance on real devices**. Each slice has an acceptance clause above; it must be demonstrated on an actual iPhone (and from Slice 9 onward, an actual Apple Watch) before the slice is considered done. Record a short screen capture in the PR description.

At the end of the project (post-Slice 10), additional end-to-end validation:

- Two-user flow: create → invite → accept → claim → assign → complete → graveyard search finds the item.
- Snooze + unsnooze across an 8-hour window (verify on real device over a wall-clock day).
- Notification delivery across all four triggers (created, claimed, assigned-to-you, completed), on both iPhone and Watch.
- Backup restore: destroy the droplet, provision fresh, restore from the latest SQLite backup, confirm app recovers.
