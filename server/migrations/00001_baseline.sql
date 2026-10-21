-- +goose Up
-- +goose StatementBegin
-- Baseline migration. Real tables land in Slice 2 and beyond.
-- Having a baseline file lets goose initialize its goose_db_version table
-- and lets Go embed *.sql without requiring at least one match at compile time.
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
