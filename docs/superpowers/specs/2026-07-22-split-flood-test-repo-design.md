# Split flood-test/PFC material into its own repository

## Problem

The `gaardrail` repo today mixes two things: the actual back-pressure-controller
software (`app/`, `cmd/`, `internal/`, `pkg/`, `web/`, CI) and everything
related to the PFC (thesis) validation rig — flood-test infra, dashboards,
scripts, VPS deploy, MATLAB modeling, academic docs. This makes the repo
harder to reason about and mixes a production-leaning artifact (the
controller) with a research artifact (the experiment rig).

## Goal

Split into two git repositories, each with its own preserved history:

- **`gaardrail`** — the controller software only. Stays as the current
  repo/remote (`git@github.com:adrianozp/gaardrail.git`), rewritten in place
  once verified.
- **`gaardrail-flood-test`** — the PFC validation rig: flood-test infra,
  Grafana dashboards, load/experiment scripts, VPS deploy config, MATLAB
  modeling, PFC docs. New repo, no remote created yet.

Both repos are created fresh as sibling directories under
`/home/adrianozdp/workspace/pfc/`, derived from `git-filter-repo` runs against
a throwaway clone of the current repo. **The current `gaardrail/` working
directory is left completely untouched** until the user has reviewed both
outputs and explicitly decides to swap.

## Non-goals

- No push to GitHub in this pass — both repos stay local until reviewed.
- No creation of a new GitHub remote for `gaardrail-flood-test`.
- No code changes to the Go application. The disturbance/floodmessage
  handlers and pid/smith controllers stay exactly as they are — they belong
  to the core repo (see rationale below).
- No attempt to deduplicate `deploy/config/config.yaml` and
  `config/config.vps` (currently near-identical) — that's a follow-up, not
  part of this split.

## Path classification

### `gaardrail` (core)

```
app/ cmd/ internal/ pkg/ web/
Dockerfile  go.mod  go.sum  LICENSE  README.md  openapi.yaml
docker-compose.yml            (kafka only, local dev)
config/config.yaml            (default local dev config)
Makefile                      (trimmed — see below)
.github/workflows/ci.yml
.gitignore  .vscode/
docs/superpowers/             (design specs/plans for the software itself,
                                including disturbance/flood-endpoint design
                                docs — these describe app code, not the rig)
```

Dropped entirely: `docs/openapi.yaml` (stale, diverging duplicate of the root
`openapi.yaml` — verified via diff, safe to delete rather than move).

Makefile after trim keeps only: `kafka/up`, `kafka/down`, `kafka/setup`,
`run`, `docker/build`. The `flood/*` targets move to the new repo's Makefile.

**Rationale for keeping disturbance/floodmessage code in core:** despite the
naming, `app/disturbance/`, `app/handlers/disturbance/`,
`app/handlers/floodmessage/`, and the `pid`/`smith` controllers under
`app/repositories/controllers/` are real product code (no build-tag
isolation exists today) that ships in the binary and is exercised by the
flood-test rig over HTTP. It doesn't get split out — the flood-test repo
will consume it as a published Docker image, not as source.

### `gaardrail-flood-test` (new)

```
flood-test/                   (all: analysis/, experiments/, grafana/,
                                scripts/, docker-compose-flood.yml, my.cnf,
                                prometheus.yml, and the *.bak files)
matlab/
docs/pfc/
brainstorming/
deploy/                       (config/config.yaml, prometheus.yml, init.sql —
                                deploy of the flood-test VPS rig)
docker-compose.vps.yml
config/config.vps
config/config.yaml.bak-pidff-t60
config/config.yaml.bak-revalidacao
.github/workflows/deploy.yml
```

Plus, newly authored for this repo:
- `Makefile` with the `flood/*` targets (`flood/up`, `flood/down`,
  `flood/setup`, `flood/messages`), unchanged from today's since `flood-test/`
  keeps the same relative path inside the new repo root.
- A short `README.md` describing the rig and pointing back at `gaardrail`
  for the controller itself.
- `.gitignore` carried over from the relevant original entries (`.env`,
  editor dirs) plus anything under `flood-test/` that shouldn't be tracked
  (none currently ignored beyond the global patterns).

Untracked files today with no history (`docs/pfc/`, `flood-test/analysis/`,
`flood-test/experiments/`, `matlab/`, the `config/*.bak` files, a few new
`flood-test/scripts/*.sh`) land directly as new files in this repo — nothing
to preserve there.

### `.env`

Currently untracked (gitignored) and contains VPS secrets used by the
flood-test deploy. It is not moved by git — the user will manually recreate
it in whichever repo actually runs `docker-compose.vps.yml` going forward
(`gaardrail-flood-test`).

## Process

1. Install `git-filter-repo` locally via `pip3 install --user git-filter-repo`
   (no sudo needed, already have pip).
2. Make a throwaway plain clone of the current repo (local path, not GitHub)
   into a temp dir — twice, one per target repo — so `git filter-repo`'s
   destructive rewrite never touches the real working directory.
3. In the "core" clone: run `git filter-repo` with `--path` allowlist for
   every path listed under `gaardrail (core)` above, plus `--path-rename`
   only if needed (none anticipated — paths keep their names). Then apply
   the Makefile trim and delete `docs/openapi.yaml` as a normal commit on
   top (history-preserving edit, not part of the filter-repo rewrite).
   Move this clone to `/home/adrianozdp/workspace/pfc/gaardrail-core-new/`
   (temporary name, to avoid colliding with the live `gaardrail/` dir).
4. In the "flood-test" clone: run `git filter-repo` with `--path` allowlist
   for every path listed under `gaardrail-flood-test (new)` above. Then add
   the new Makefile and README as a commit on top. Move this clone to
   `/home/adrianozdp/workspace/pfc/gaardrail-flood-test/`.
5. Copy the untracked files (see list above) from the current working
   `gaardrail/` into the correct new repo, `git add` + commit them there.
6. Report both new repos to the user with `git log --oneline` and `git
   status` for review. Do not touch the original `gaardrail/` directory, do
   not push anything, do not create a GitHub remote for the new repo.
7. Once the user confirms both look right, a follow-up step (outside this
   spec) will swap `gaardrail-core-new/` into place at `gaardrail/` and the
   user decides when to push.

## Verification

- Each new repo: `go build ./...` / `go test ./...` runs clean in the core
  repo (proves nothing load-bearing got dropped).
  the flood-test repo has no Go code, so verification there is: docker
  compose config validates (`docker compose -f docker-compose-flood.yml
  config`, `docker compose -f docker-compose.vps.yml config`) and the
  Makefile targets reference paths that actually exist.
- `git log --stat` spot-check on a couple of known-old files (e.g.
  `app/repositories/controllers/pid/pid.go`, `flood-test/scripts/*`) in each
  new repo to confirm history was preserved, not squashed.
- Diff the working tree of each new repo against the relevant subtree of the
  original `gaardrail/` to confirm no content was lost or altered
  unexpectedly (aside from the deliberate Makefile trim and
  `docs/openapi.yaml` removal).
