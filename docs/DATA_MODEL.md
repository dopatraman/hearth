# Hearth — Data Model

Reference for every slice. If you're about to touch persistence, read this first.

## Database engine

**SQLite** via `modernc.org/sqlite` (pure Go, no CGo).

SQLite is a fine fit for Hearth at household scale: one household, 2–6 users, a few writes per minute at peak. SQLite serializes writers (one writer transaction at a time); this is a non-issue at our scale. Chose it over Postgres to eliminate daemon/role/backup/cost overhead on the droplet. If "open to multiple households" is ever a goal, this is the piece that needs revisiting.

**Per-connection pragmas** (set in both the application and the test harness):

```
PRAGMA foreign_keys = ON;   -- actually enforce foreign keys (off by default)
PRAGMA journal_mode = WAL;  -- better reader/writer concurrency
PRAGMA busy_timeout = 5000; -- 5s before returning "database is locked"
```

## Conventions

- Primary keys are **UUIDv7** generated application-side (Go's `github.com/google/uuid` with `uuid.NewV7()`), stored as `TEXT`. UUIDv7 is time-ordered → good index locality and rough sortability without a separate `created_at` sort key.
- Timestamps are stored as `TEXT` in ISO-8601 UTC (e.g., `"2026-10-21T04:17:00.123Z"`). SQLite doesn't have a native datetime type; we use text so SQLite's `strftime` / `datetime` functions work and so Go's `time.Time` round-trips cleanly via `time.RFC3339Nano`.
- Enum-ish columns use `TEXT` with a `CHECK` constraint listing allowed values.
- Booleans are `INTEGER` (`0` or `1`) — SQLite's convention.
- All tables carry `created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))` unless noted. Tables that are mutated also carry `updated_at` with a trigger that bumps it on `UPDATE`.
- Foreign keys default to `ON DELETE RESTRICT` unless noted. Enforcement is only active when `PRAGMA foreign_keys = ON` is set per connection — see the pragma block above.
- No ORM. SQL is the schema of record (`server/migrations/`). `sqlc` reads the schema and generates typed Go functions from queries (`server/queries/`).

## Entities

### `users`

Represents a person signed in via Sign in with Apple.

| Column | Type | Notes |
|---|---|---|
| `id` | `TEXT` PK | UUIDv7, app-generated. |
| `apple_sub` | `TEXT` NOT NULL UNIQUE | Apple's stable per-app user identifier (`sub` claim in Apple's JWT). Primary natural key. |
| `display_name` | `TEXT` | Apple only provides this on *first* sign-in; may be NULL for returning users who signed in on another device first. |
| `notifications_enabled` | `INTEGER` NOT NULL DEFAULT `1` CHECK (`notifications_enabled IN (0,1)`) | Global on/off boolean. Per-event toggles are post-MVP. |
| `created_at` | `TEXT` | |
| `updated_at` | `TEXT` | |

**Invariants**:
- `apple_sub` is immutable after creation.

### `lists`

A shared household list. For MVP, every user belongs to exactly one list, but the schema is already many-to-many (see `list_memberships`) so adding multi-list is additive.

| Column | Type | Notes |
|---|---|---|
| `id` | `TEXT` PK | UUIDv7. |
| `name` | `TEXT` NOT NULL DEFAULT `'Home'` | Shown in UI when multi-list arrives; harmless to carry now. |
| `created_at` | `TEXT` | |
| `updated_at` | `TEXT` | |

### `list_memberships`

Join table between `users` and `lists`.

| Column | Type | Notes |
|---|---|---|
| `list_id` | `TEXT` NOT NULL REFERENCES `lists(id)` ON DELETE CASCADE | Part of composite PK. |
| `user_id` | `TEXT` NOT NULL REFERENCES `users(id)` ON DELETE CASCADE | Part of composite PK. |

**Primary key**: composite `(list_id, user_id)`. Guarantees a user can't be a member of the same list twice.

**MVP constraint (enforced in application code, not schema)**: a user has exactly one row here. The schema already allows many so no migration is needed when multi-list ships.

### `chores`

A chore. Lives on exactly one list, visible to every member of that list.

| Column | Type | Notes |
|---|---|---|
| `id` | `TEXT` PK | UUIDv7. |
| `list_id` | `TEXT` NOT NULL REFERENCES `lists(id)` ON DELETE CASCADE | Which list it belongs to. |
| `creator_id` | `TEXT` NOT NULL REFERENCES `users(id)` | **Only user who can edit or delete this chore.** |
| `title` | `TEXT` NOT NULL | |
| `notes` | `TEXT` | Optional free text. |
| `state` | `TEXT` NOT NULL DEFAULT `'open'` CHECK (`state IN ('open','completed')`) | Hard-deleted chores don't exist here; completed chores persist (graveyard). |
| `claimed_by` | `TEXT` REFERENCES `users(id)` | NULL if unclaimed. |
| `claimed_at` | `TEXT` | NULL if unclaimed. |
| `will_complete_by` | `TEXT` | Set by the claimant. |
| `completed_by` | `TEXT` REFERENCES `users(id)` | NULL unless `state = 'completed'`. |
| `completed_at` | `TEXT` | NULL unless `state = 'completed'`. |
| `created_at` | `TEXT` | |
| `updated_at` | `TEXT` | |

**Invariants** (enforced in application code):

1. `claimed_by IS NULL` ⇔ `claimed_at IS NULL` ⇔ `will_complete_by IS NULL`. All or nothing: a chore is either unclaimed (all three null) or claimed (first two non-null, third optional).
2. `state = 'completed'` ⇒ `completed_by IS NOT NULL AND completed_at IS NOT NULL`.
3. Only `creator_id` may issue PATCH/DELETE on title/notes. Enforced at the API layer.
4. Setting `will_complete_by` on an unclaimed chore **implicitly claims it** for the caller (atomic in the same transaction).
5. Claiming a chore that is already claimed by someone else returns a conflict (409). Enforced by `UPDATE ... WHERE id = ? AND claimed_by IS NULL` and checking affected rows.
6. Completing a chore is only allowed by the current `claimed_by` user. Enforced at the API layer.

**Delete semantics**:
- **Creator deletes an open chore** → hard DELETE from `chores`. Snoozes referencing it cascade (`ON DELETE CASCADE`).
- **Completed chores** → remain in table with `state = 'completed'`. Searchable via graveyard query. Never hard-deleted.

### `snoozes`

Per-user per-chore snooze state.

| Column | Type | Notes |
|---|---|---|
| `user_id` | `TEXT` NOT NULL REFERENCES `users(id)` ON DELETE CASCADE | Part of composite PK. |
| `chore_id` | `TEXT` NOT NULL REFERENCES `chores(id)` ON DELETE CASCADE | Part of composite PK. If the chore is hard-deleted, the snooze goes with it. |
| `snoozed_until` | `TEXT` NOT NULL | ISO-8601 UTC. 8h from now at write time for MVP. |

**Primary key**: composite `(user_id, chore_id)`. **Upsert pattern** (`INSERT ... ON CONFLICT (user_id, chore_id) DO UPDATE SET snoozed_until = excluded.snoozed_until`) — re-snoozing updates `snoozed_until` in place; does not append a second row. One row per (user, chore) pair, forever bounded.

**Invariants**:
- No background job needed to expire snoozes. Queries filter `snoozed_until <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now')` and treat expired rows as "not snoozed." Rows can live indefinitely; harmless.
- Snoozes are per-user. Alice snoozing a chore does **not** affect Bob's view.
- Snooze is only meaningful on the personal list query. Backlog ignores snoozes.

### `events`

Append-only audit log. Drives notifications and the future activity feed. Never updated, never deleted.

| Column | Type | Notes |
|---|---|---|
| `id` | `TEXT` PK | UUIDv7 (natural chronological ordering). |
| `list_id` | `TEXT` NOT NULL REFERENCES `lists(id)` ON DELETE CASCADE | For scoping feed queries. |
| `chore_id` | `TEXT` NOT NULL REFERENCES `chores(id)` ON DELETE CASCADE | All MVP events are chore-related. If chore is hard-deleted, events go with it. |
| `actor_id` | `TEXT` NOT NULL REFERENCES `users(id)` | Who did the thing. |
| `type` | `TEXT` NOT NULL CHECK (`type IN (...)`) | See allowed values below. |
| `created_at` | `TEXT` | |

**Allowed `type` values** (MVP, current migrations):
- `created` — chore created
- `claimed` — chore claimed by actor
- `unclaimed` — actor relinquished their claim
- `will_complete_by_set` — actor set / updated `will_complete_by`
- `snoozed` — actor snoozed the chore on their personal view
- `completed` — chore marked completed by actor
- `edited` — creator edited the title/notes

**Invariants**:
- Events are written in the **same transaction** as the state change they describe. If the write fails, both roll back together.
- No `UPDATE` or `DELETE` from application code. Any cleanup is a future admin operation.

### `device_tokens`

APNs device tokens for push notification fan-out.

| Column | Type | Notes |
|---|---|---|
| `user_id` | `TEXT` NOT NULL REFERENCES `users(id)` ON DELETE CASCADE | Part of composite PK. |
| `token` | `TEXT` NOT NULL | The APNs device token (hex string). Part of composite PK. |
| `platform` | `TEXT` NOT NULL CHECK (`platform IN ('ios','watchos')`) | For routing payloads and tracking per-device delivery. |
| `created_at` | `TEXT` | |

**Primary key**: composite `(user_id, token)`. The same device registering twice is a no-op.

**Invariants**:
- When APNs returns `410 Gone` for a token, delete that row.

### `invites`

Short-lived codes a user can share to let someone else join their list.

| Column | Type | Notes |
|---|---|---|
| `code` | `TEXT` PK | 6-digit code, zero-padded (e.g., `'428193'`). |
| `list_id` | `TEXT` NOT NULL REFERENCES `lists(id)` ON DELETE CASCADE | Which list this invite grants access to. |
| `created_by` | `TEXT` NOT NULL REFERENCES `users(id)` | For auditing / display ("Alice invited you"). |
| `created_at` | `TEXT` | |
| `expires_at` | `TEXT` NOT NULL | `created_at + 24h`. |

**Invariants**:
- Codes are **multi-use within their 24h window** — one code can let multiple people join (useful for inviting a household at once). Redemption history lives in `list_memberships` and `events`, not here.
- Expired codes return `410 Gone` on accept. Nonexistent codes return `404 Not Found`.
- Code generation is `random 6 digits`, retry on PK collision. 1M space, 24h TTL — collisions rare but possible, handle them.

## Relationships (quick sketch)

```
users ─┬─< list_memberships >─── lists ──< chores
       │                                     │
       ├─< device_tokens                     ├─< snoozes >─ users
       │                                     │
       ├── creator ─────────────────────────┤
       ├── claimed_by (nullable) ───────────┤
       ├── completed_by (nullable) ─────────┤
       │                                     │
       └── actor ─< events >─────────────────┘
                   list_id ─────────────────── lists
```

## Indexes

Required for MVP, declared in migrations alongside the tables they index:

| Index | Purpose |
|---|---|
| `CREATE INDEX chores_list_open ON chores (list_id) WHERE state = 'open';` | Hot path: backlog query. Partial index keeps it small. SQLite supports partial indexes natively. |
| `CREATE INDEX snoozes_user_until ON snoozes (user_id, snoozed_until);` | Fast filter for personal-list query. |
| `CREATE INDEX events_list_time ON events (list_id, created_at DESC);` | Feeds, activity log. |
| `CREATE INDEX events_chore_time ON events (chore_id, created_at);` | Per-chore history. |
| `CREATE INDEX list_memberships_user ON list_memberships (user_id);` | "What lists does this user belong to?" |
| `CREATE INDEX device_tokens_user ON device_tokens (user_id);` | Push fan-out. |
| `CREATE INDEX invites_expires ON invites (expires_at);` | Cleanup of expired codes. |

Automatic indexes for primary keys and `UNIQUE` constraints are not listed above.

## Planned / post-MVP

Not yet in migrations; captured here so the shape is clear when their slices arrive.

- **FTS5 full-text search on chores**. SQLite's FTS5 extension is built in. Plan: shadow virtual table (`chores_fts`) carrying `rowid, title, notes` and tied to `chores` via `INSERT`/`UPDATE`/`DELETE` triggers. Searches use `chores_fts MATCH 'trash'` and join back to `chores`. Trigram tokenizer for substring, prefix tokenizer for dedup suggestions on new-chore flow. Added when dedup search and graveyard search are built.
- **Granular notification preferences**: add a `user_notification_prefs(user_id, event_type, enabled)` table alongside the global `users.notifications_enabled`.
- **Recurrence**: `recurrence_rules(chore_id, rrule, next_occurrence_at)` + nullable `recurrence_rule_id` on chores.
- **Assignment**: add `chores.assigned_by` (nullable FK to users) + an `assigned` event type. Current migrations don't have either; both land together when assignment ships.
- **Nudge / force-unclaim**: `nudges(chore_id, nudger_id, nudged_user_id, created_at)`.
- **List owner / kick**: `list_memberships.role` enum.

## Cross-entity invariants

These hold across tables and are enforced in application code, usually inside a single transaction:

1. **Event-atomicity**: every write to `chores` (create, claim, unclaim, will_complete_by, complete, edit) writes a corresponding row to `events` in the same transaction. If one fails, both roll back.
2. **Membership-scoping**: every query that returns chores filters by `list_id IN (SELECT list_id FROM list_memberships WHERE user_id = ?)`. The API layer is responsible for this; there is no row-level security in SQLite.
3. **Snooze cleanup on hard-delete**: hard-deleting a `chores` row cascades to `snoozes` (handled by `ON DELETE CASCADE`, which requires `PRAGMA foreign_keys = ON`).
4. **Device token lifecycle**: on APNs `410 Gone`, delete the token row.

## MVP ⇒ post-MVP migration notes

Things the schema was built to allow without migrations:

- **Multi-list per user**: already modeled. Add a list-picker in the UI.
- Everything else that would need a schema change is listed in **Planned / post-MVP** above.

If **multi-household** becomes a goal, SQLite is where this gets interesting: single-writer serialization is fine for one household but will bite at scale. The move at that point is a one-time dump → Postgres restore, swap the driver and sqlc engine, keep the schema roughly as-is (dialect differences on partial indexes and FTS are the main things to port).
