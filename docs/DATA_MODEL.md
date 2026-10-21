# Hearth — Data Model

Reference for every slice. If you're about to touch persistence, read this first.

## Conventions

- All primary keys are UUIDs. Prefer `uuid_v7` where available (time-ordered → better btree locality + rough sortability). Where `uuid_v4` is used, it's noted.
- All timestamps are `TIMESTAMPTZ`, stored in UTC. Clients render local.
- All tables carry `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`. Tables that are mutated (not append-only) also carry `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`, maintained by a trigger or application code.
- Foreign keys default to `ON DELETE RESTRICT` unless noted. The one exception: when a user is hard-deleted (future feature, not MVP), we'll need to think about cascade carefully — deferred.
- Enum-ish columns use `TEXT` with a `CHECK` constraint rather than native Postgres `ENUM` — easier to add values via migration without rewriting the type.
- No ORM. SQL is the schema of record (`server/migrations/`). `sqlc` reads the schema and generates typed Go functions from queries (`server/queries/`).

## Entities

### `users`

Represents a person signed in via Sign in with Apple.

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` PK | uuid_v7 |
| `apple_sub` | `TEXT` NOT NULL UNIQUE | Apple's stable per-app user identifier (`sub` claim in Apple's JWT). Primary natural key. |
| `display_name` | `TEXT` | Apple only provides this on *first* sign-in; may be NULL for returning users who signed in on another device first. |
| `notifications_enabled` | `BOOLEAN` NOT NULL DEFAULT `TRUE` | Global on/off for MVP. Per-event toggles are post-MVP. |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

**Invariants**:
- `apple_sub` is immutable after creation. If Apple's claim changes (it doesn't), investigate before updating.

### `lists`

A shared household list. For MVP, every user belongs to exactly one list, but the schema is already many-to-many (see `list_memberships`) so adding multi-list is additive.

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` PK | uuid_v7 |
| `name` | `TEXT` NOT NULL DEFAULT `'Home'` | Shown in UI when multi-list arrives; harmless to carry now. |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

### `list_memberships`

Join table between `users` and `lists`.

| Column | Type | Notes |
|---|---|---|
| `list_id` | `UUID` NOT NULL REFERENCES `lists(id)` | Part of composite PK. |
| `user_id` | `UUID` NOT NULL REFERENCES `users(id)` | Part of composite PK. |
| `joined_at` | `TIMESTAMPTZ` NOT NULL DEFAULT `now()` | |

**Primary key**: composite `(list_id, user_id)`. Guarantees a user can't be a member of the same list twice.

**MVP constraint (enforced in application code, not schema)**: a user has exactly one row here. The schema already allows many so no migration is needed when multi-list ships.

### `items`

A chore. Lives on exactly one list, visible to every member of that list.

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` PK | uuid_v7 |
| `list_id` | `UUID` NOT NULL REFERENCES `lists(id)` | Which list it belongs to. |
| `creator_id` | `UUID` NOT NULL REFERENCES `users(id)` | **Only user who can edit or delete this item.** |
| `title` | `TEXT` NOT NULL | Indexed for trigram search (see Indexes). |
| `notes` | `TEXT` | Optional free text. |
| `state` | `TEXT` NOT NULL DEFAULT `'open'` CHECK (`state IN ('open','completed')`) | Hard-deleted items don't exist here; completed items persist (graveyard). |
| `claimed_by` | `UUID` REFERENCES `users(id)` | NULL if unclaimed. |
| `claimed_at` | `TIMESTAMPTZ` | NULL if unclaimed. |
| `assigned_by` | `UUID` REFERENCES `users(id)` | NULL if self-claimed. Non-null means "X assigned this to the claimant." |
| `will_complete_by` | `TIMESTAMPTZ` | Set by the claimant (or set at assignment time by the assigner, implying implicit claim — see Invariants). |
| `completed_by` | `UUID` REFERENCES `users(id)` | NULL unless `state = 'completed'`. |
| `completed_at` | `TIMESTAMPTZ` | NULL unless `state = 'completed'`. |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

**Invariants** (enforced in application code; a few can also be CHECK constraints):

1. `claimed_by IS NULL` ⇔ `claimed_at IS NULL` ⇔ `will_complete_by IS NULL` ⇔ `assigned_by IS NULL`. All or nothing: an item is either unclaimed (all four null) or claimed (first two non-null, last two optional).
2. `state = 'completed'` ⇒ `completed_by IS NOT NULL AND completed_at IS NOT NULL`.
3. Only `creator_id` may issue PATCH/DELETE on title/notes. Enforced at the API layer.
4. Setting `will_complete_by` on an unclaimed item **implicitly claims it** for the caller (atomic in the same transaction).
5. Claiming an item that is already claimed by someone else returns a conflict (409). Enforced by `UPDATE ... WHERE id = $1 AND claimed_by IS NULL` and checking affected rows.
6. Completing an item is only allowed by the current `claimed_by` user. Enforced at the API layer.

**Delete semantics**:
- **Creator deletes an open item** → hard DELETE from `items`. Any snoozes referencing it must `ON DELETE CASCADE` (see snoozes).
- **Completed items** → remain in table with `state = 'completed'`. Searchable via graveyard query. Never hard-deleted.

### `snoozes`

Per-user per-item snooze state. See [the plan](plans/BUILD_PLAN.md) for the extended rationale.

| Column | Type | Notes |
|---|---|---|
| `user_id` | `UUID` NOT NULL REFERENCES `users(id)` ON DELETE CASCADE | Part of composite PK. |
| `item_id` | `UUID` NOT NULL REFERENCES `items(id)` ON DELETE CASCADE | Part of composite PK. If the item is hard-deleted, the snooze goes with it. |
| `snoozed_until` | `TIMESTAMPTZ` NOT NULL | `now() + 8h` for MVP (no picker). |
| `created_at` | `TIMESTAMPTZ` | |

**Primary key**: composite `(user_id, item_id)`. **Upsert pattern** — re-snoozing updates `snoozed_until` in place; does not append a second row. One row per (user, item) pair, forever bounded.

**Invariants**:
- No background job needed to expire snoozes. Queries filter `snoozed_until <= now()` and treat expired rows as "not snoozed." Rows can live indefinitely; harmless.
- Snoozes are per-user. Alice snoozing an item does **not** affect Bob's view.
- Snooze is only meaningful on the personal list query. Backlog ignores snoozes.

### `events`

Append-only audit log. Drives notifications in Slice 8 and the future activity feed. Never updated, never deleted.

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` PK | uuid_v7 (natural chronological ordering). |
| `list_id` | `UUID` NOT NULL REFERENCES `lists(id)` | For scoping feed queries. |
| `item_id` | `UUID` NOT NULL REFERENCES `items(id)` | All MVP events are item-related. If item is hard-deleted, events go with it (`ON DELETE CASCADE`). |
| `actor_id` | `UUID` NOT NULL REFERENCES `users(id)` | Who did the thing. |
| `target_user_id` | `UUID` REFERENCES `users(id)` | For `assigned` events: the assignee. NULL for other types. |
| `type` | `TEXT` NOT NULL CHECK (`type IN (...)`) | See allowed values below. |
| `payload` | `JSONB` | Type-specific details. Optional. |
| `created_at` | `TIMESTAMPTZ` | |

**Allowed `type` values** (MVP):
- `created` — item created
- `claimed` — item claimed by actor
- `unclaimed` — actor relinquished their claim
- `assigned` — actor assigned the item to `target_user_id`
- `will_complete_by_set` — actor set / updated `will_complete_by`
- `snoozed` — actor snoozed the item on their personal view
- `completed` — item marked completed by actor
- `edited` — creator edited the title/notes

**Invariants**:
- Events must be written in the **same transaction** as the state change they describe. If the write fails, both roll back together.
- No `UPDATE` or `DELETE` from application code. Any cleanup is a future admin operation.

### `device_tokens`

APNs device tokens for push notification fan-out. One row per unique `(user_id, token)` combination.

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` PK | |
| `user_id` | `UUID` NOT NULL REFERENCES `users(id)` ON DELETE CASCADE | |
| `token` | `TEXT` NOT NULL | The APNs device token (hex string). |
| `platform` | `TEXT` NOT NULL CHECK (`platform IN ('ios','watchos')`) | For routing payloads and tracking per-device delivery. |
| `created_at` | `TIMESTAMPTZ` | |
| `last_seen_at` | `TIMESTAMPTZ` NOT NULL DEFAULT `now()` | Updated on each successful push. |

**Primary unique**: `UNIQUE (user_id, token)` so the same device registering twice is a no-op.

**Invariants**:
- When APNs returns `410 Gone` for a token, delete that row (see Slice 8).

### `invites`

Short-lived codes a user can share to let someone else join their list. Added in Slice 4.

| Column | Type | Notes |
|---|---|---|
| `code` | `TEXT` PK | 6-digit code, zero-padded (e.g., `'428193'`). Collisions are rejected on insert; caller retries. |
| `list_id` | `UUID` NOT NULL REFERENCES `lists(id)` ON DELETE CASCADE | Which list this invite grants access to. |
| `created_by` | `UUID` NOT NULL REFERENCES `users(id)` | For auditing / display ("Alice invited you"). |
| `created_at` | `TIMESTAMPTZ` | |
| `expires_at` | `TIMESTAMPTZ` NOT NULL | `created_at + 24h`. |

**Invariants**:
- Codes are **multi-use within their 24h window** — one code can let multiple people join (useful for inviting a household at once). Redemption history lives in `list_memberships.joined_at` and `events`, not here.
- Expired codes return `410 Gone` on accept. Nonexistent codes return `404 Not Found`.
- Code generation is `random 6 digits`, retry on PK collision. 1M space, 24h TTL — collisions are rare but possible, handle them.

## Relationships (quick sketch)

```
users ─┬─< list_memberships >─── lists ──< items
       │                                     │
       ├─< device_tokens                     ├─< snoozes >─ users
       │                                     │
       ├── creator ─────────────────────────┤
       ├── claimed_by (nullable) ───────────┤
       ├── assigned_by (nullable) ──────────┤
       ├── completed_by (nullable) ─────────┤
       │                                     │
       └── actor ─< events >─────────────────┘
                   target_user_id (nullable) → users
                   list_id ─────────────────── lists
```

## Indexes

Required for MVP, declared in migrations alongside the tables they index:

| Index | Purpose |
|---|---|
| `CREATE EXTENSION IF NOT EXISTS pg_trgm;` | Enables trigram GIN indexes for substring search. |
| `CREATE INDEX items_title_trgm ON items USING GIN (title gin_trgm_ops);` | Dedup search when creating new items + graveyard search. |
| `CREATE INDEX items_list_open ON items (list_id) WHERE state = 'open';` | Hot path: backlog query. Partial index keeps it small. |
| `CREATE INDEX snoozes_user_until ON snoozes (user_id, snoozed_until);` | Fast filter for personal-list query. |
| `CREATE INDEX events_list_time ON events (list_id, created_at DESC);` | Feeds, activity log. |
| `CREATE INDEX events_item_time ON events (item_id, created_at);` | Per-item history. |
| `CREATE INDEX list_memberships_user ON list_memberships (user_id);` | "What lists does this user belong to?" |
| `CREATE INDEX device_tokens_user ON device_tokens (user_id);` | Push fan-out. |
| `CREATE INDEX invites_expires ON invites (expires_at);` | Cleanup of expired codes. |

The automatic indexes for primary keys and `UNIQUE` constraints are not listed above.

## Cross-entity invariants

These hold across tables and are enforced in application code, usually inside a single transaction:

1. **Event-atomicity**: every write to `items` (create, claim, unclaim, assign, will_complete_by, complete, edit) writes a corresponding row to `events` in the same transaction. If one fails, both roll back.
2. **Membership-scoping**: every query that returns items filters by `list_id IN (SELECT list_id FROM list_memberships WHERE user_id = $current)`. The API layer is responsible for this; there is no row-level security configured in Postgres.
3. **Snooze cleanup on hard-delete**: hard-deleting an `items` row cascades to `snoozes` (handled by `ON DELETE CASCADE`).
4. **Device token lifecycle**: on APNs `410 Gone`, delete the token row. On successful push, bump `last_seen_at`.

## MVP ⇒ post-MVP migration notes

Things we built the schema to allow without migrations:

- **Multi-list per user**: already modeled. Add a list-picker in the UI.
- **Granular notification preferences**: add a `user_notification_prefs(user_id, event_type, enabled)` table alongside the global `users.notifications_enabled` boolean.
- **Recurrence**: add a `recurrence_rules(item_id, rrule, next_occurrence_at)` table and nullable `recurrence_rule_id` on items. Deliberately deferred.
- **Nudge / force-unclaim**: add an `nudges(item_id, nudger_id, nudged_user_id, created_at)` table. Deliberately deferred.
- **List owner / kick**: add `list_memberships.role` enum. Deliberately deferred.
