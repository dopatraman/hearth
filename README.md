# Hearth

Shared household chore tracking for iOS 26+ and Apple Watch.

Early work in progress. See [docs/plans/BUILD_PLAN.md](docs/plans/BUILD_PLAN.md) for the build plan and [docs/DATA_MODEL.md](docs/DATA_MODEL.md) for the data model.

## Layout

- `server/` — Go API (chi + pgx + sqlc, Postgres)
- `apps/` — Xcode workspace with iOS and watchOS targets
- `deploy/` — docker-compose, Caddyfile, droplet provisioning
- `docs/` — plans and reference docs

## Development

```sh
just dev      # bring up Postgres + run server locally
just test     # run all tests (Go + iOS)
just migrate  # apply pending migrations
```
