# Contributing

Small repo, simple rules.

## Branch + PR

- Branch off `main`. Name with intent: `feat/<scope>`, `fix/<scope>`,
  `docs/<scope>`, `refactor/<scope>`.
- Open a PR against `main`. CI must pass (build + vet + test + golangci-lint).
- One reviewer approval before merge. Squash on merge if the branch has noisy fixup commits.

## Commit messages

Subject in present tense, ≤ 70 chars. Body explains the *why* — what was
broken / what constraint forced the choice. Wrap at 72.

```
fix(ratelimit): per-IP bucket leaks under x-forwarded-for spoof

x-forwarded-for can be set by any client when there's no fronting proxy.
Trust it only when the trusted-proxy list contains the immediate peer;
otherwise fall back to the TCP peer addr.
```

Do not amend pushed commits. New commits, always.

## Code style

- `make fmt` (gofumpt + goimports with the local-prefix set in `.golangci.yml`).
- `make lint` (golangci-lint with the shared config). Both run in CI.
- Keep `pkg/` business-agnostic. If a change to `pkg/` is motivated by one
  service's needs, justify in the PR body why it generalizes.

## Tests

- `make test-all` must pass with `-race -count=1`.
- New `pkg/` code should ship with at least one behavioral test (see
  `pkg/middleware/*/`*_test.go` for the pattern).
- Use `pkg/testutil` for middleware fakes; don't roll your own.

## Adding a new service

Use the scaffolder:

```bash
make new-service NAME=worker GRPC=50054 HTTP=8086 OTHER=8087
```

This wires everything (cmd/, Makefile, Dockerfile, helm-chart, go.work entry,
port substitutions, identifier rewrites) so the diff is small and reviewable.

## Bumping pinned tools

Versions are centralized in `tools/install.sh`. Bump there only; per-service
`make init` uses `@latest` for ergonomics, but CI follows the pinned set.
