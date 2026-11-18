-- +goose Up

-- +goose StatementBegin
ALTER TABLE chores ADD COLUMN claimed_order INTEGER;
-- +goose StatementEnd

-- Backfill: for every already-claimed open chore, assign a dense per-list
-- rank that mirrors the client's current sort of the Claimed section:
-- overdue first, then nearest will_complete_by, then oldest-created.
-- Keeps the Backlog visually stable for returning users on first boot.
-- +goose StatementBegin
WITH ranked AS (
    SELECT c.id,
           row_number() OVER (
               PARTITION BY c.list_id
               ORDER BY (
                   CASE
                       WHEN c.will_complete_by IS NOT NULL
                        AND c.will_complete_by < strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
                       THEN 1 ELSE 0
                   END
               ) DESC,
               c.will_complete_by IS NULL,
               c.will_complete_by ASC,
               c.created_at ASC
           ) AS rnk
      FROM chores c
     WHERE c.state = 'open'
       AND c.claimed_at IS NOT NULL
)
UPDATE chores
   SET claimed_order = (SELECT rnk FROM ranked WHERE ranked.id = chores.id)
 WHERE id IN (SELECT id FROM ranked);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX chores_list_claimed_order
    ON chores (list_id, claimed_order)
    WHERE claimed_order IS NOT NULL;
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS chores_list_claimed_order;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE chores DROP COLUMN claimed_order;
-- +goose StatementEnd
