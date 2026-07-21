# Contributing

Thanks for your interest in contributing to Pulse!

## Getting started

1. Fork and clone the repository.
2. Create a feature branch: `git checkout -b feature/short-description`.
3. Follow [docs/setup.md](docs/setup.md) to get the stack running locally.

## Repository layout

- `api/common` — Go/Gin backend (auth + core API)
- `web` — Next.js web app + dashboard
- `docs`, `scripts`

## Naming conventions

- Go: lowercase, singular package names (`handler`, `service`, `model`); `snake_case.go` files.
- Next/TS: kebab-case folders (`change-password/`); **components PascalCase** (`NavBar.tsx`);
  modules/hooks/styles/types kebab-case (`use-auth.ts`); Next reserved files stay lowercase
  (`page.tsx`, `layout.tsx`, `route.ts`).
  Existing components predate this convention and are kebab-case (`sidebar.tsx`) — that's fine
  as-is; apply PascalCase to **new** component files going forward rather than mass-renaming.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):
`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`.

## Before opening a pull request

- Run the relevant lint/build for the area you changed (`go vet ./...` and `go build ./...`
  in `api/common`; `npm run build` / `npx tsc --noEmit` in `web`).
- Do **not** commit secrets, credentials, or `.env` files.
- Fill out the pull request template and link any related issues.
- Keep PRs focused; smaller PRs review faster.

## Code review

At least one approving review from a [CODEOWNER](CODEOWNERS) is required before merge.

## Code of Conduct

Participation is governed by our [Code of Conduct](CODE_OF_CONDUCT.md).
