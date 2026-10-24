# Private lists — verification

## Automated checks

From `backend/`:

```sh
go test -race ./...
golangci-lint run ./...
go vet ./...
sqlc generate
```

From `frontend/`:

```sh
pnpm check
pnpm test --run
pnpm test:e2e
pnpm build
```

Browser tests use dedicated frontend/backend ports 5183/8081, an in-memory
database, and virtual passkeys. They must not reuse the developer's running app
or database. Fresh-browser coverage seeds only a passkey, not cookies, local
storage, or IndexedDB list data.

## Coverage

- Migration up/down, foreign keys, owner filtering, and item counts.
- Valid, missing, malformed, expired, and revoked sessions.
- Ownership checks on every list/item operation, including cross-list item IDs.
- Cross-origin write rejection and `no-store` private responses.
- Item creation, editing, check/uncheck, deletion, and atomic clear-checked.
- Independent field changes merge; same-field changes follow backend order.
- Check/uncheck uses explicit states and backend timestamps; text/department edits
  preserve the checked state.
- Concurrent additions all succeed; retries of the same addition ID do not
  duplicate or overwrite it, including when its response was lost.
- Deletion wins over edits in either processing order; late edits are no-ops and
  retried additions cannot resurrect tombstones.
- Clear checked uses backend state at processing time (uncheck-before-clear and
  clear-before-uncheck have the documented different outcomes).
- Clients send no version preconditions and do not handle revision conflicts.
  Failed transactions roll back both data changes and list timestamps.
- Database close/reopen preserves sessions, lists, items, and tombstones.
- Stored timestamps use integer Unix seconds; API timestamps remain ISO strings.
- Forward/backward migration preserves existing lists, items, and tombstones.
- Name/text limits are checked before writes, including exact boundaries and
  non-ASCII text; oversized values return 400 without modifying the list.
- Browser reload and independent sign-in in a fresh browser preserve lists.
- Offline failures never apply changes, clear unsaved item input, or queue work.
- Failed swipe deletions restore the row and can be retried.
- Account changes clear memory and ignore late responses from the previous account.
- A rejected session clears private data and offers sign-in.
- Old browser list data is ignored and left intact; a fresh browser does not
  create a list database.

## Before deploying

1. Back up the SQLite database and verify the deployment uses persistent storage.
2. Build the frontend before building the backend's embedded production bundle.
3. Smoke-test sign-in, list creation, and another signed-in device against the
   production origin. Verify list persistence across a process restart.
4. Confirm the UI explains that previous browser-only lists are not imported.

Automated persistence coverage closes/reopens a file-backed database; deployment
storage, backups, and a production restart still require an operational smoke test.
No deployment is performed by these checks.
