-- +goose Up

-- +goose StatementBegin
ALTER TABLE chores RENAME COLUMN claimed_order TO section_order;
-- +goose StatementEnd

-- Backfill Up-for-grabs rows (open + unclaimed) with a dense per-list rank
-- by created_at ASC — mirrors the client's current Up-for-grabs sort so
-- returning users see no visual jump. Claimed rows already carry their
-- values from migration 00006.
-- +goose StatementBegin
WITH ranked AS (
    SELECT c.id,
           row_number() OVER (
               PARTITION BY c.list_id
               ORDER BY c.created_at ASC
           ) AS rnk
      FROM chores c
     WHERE c.state = 'open'
       AND c.claimed_at IS NULL
)
UPDATE chores
   SET section_order = (SELECT rnk FROM ranked WHERE ranked.id = chores.id)
 WHERE id IN (SELECT id FROM ranked);
-- +goose StatementEnd

-- SQLite preserves indexes across column renames — the old partial-index
-- definition still references the renamed column by its new name. Drop
-- and recreate so the index name reflects the new column.
-- +goose StatementBegin
DROP INDEX IF EXISTS chores_list_claimed_order;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX chores_list_section_order
    ON chores (list_id, section_order)
    WHERE section_order IS NOT NULL;
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS chores_list_section_order;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX chores_list_claimed_order
    ON chores (list_id, section_order)
    WHERE section_order IS NOT NULL;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE chores RENAME COLUMN section_order TO claimed_order;
-- +goose StatementEnd
