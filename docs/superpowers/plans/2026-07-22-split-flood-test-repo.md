# Split flood-test/PFC material into its own repository — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the current `gaardrail` repo into two independent git repos with preserved history — `gaardrail` (controller software only) and `gaardrail-flood-test` (PFC validation rig) — without touching the live working directory until both are verified.

**Architecture:** Two throwaway plain clones of the current repo, each run through `git-filter-repo` with a `--path` allowlist matching the classification in the spec, then a small number of follow-up commits (Makefile trim, new README, untracked files) on top of each filtered history. Final outputs land in new sibling directories; the live `gaardrail/` repo is never modified by this plan.

**Tech Stack:** git, git-filter-repo (pip), bash, Go toolchain (verification only), Docker Compose (config validation only).

## Global Constraints

- Do not modify, rewrite, or push the current working repo (`/home/adrianozdp/workspace/pfc/gaardrail`) at any point in this plan.
- No push to any GitHub remote. No `gh repo create`.
- All destructive/rewriting git operations (`filter-repo`) run only against throwaway clones in scratch space, never against the live repo or its `.git`.
- Every `--path` given to `filter-repo` must come from the classification lists in `docs/superpowers/specs/2026-07-22-split-flood-test-repo-design.md` — no ad hoc additions.
- Untracked files (no git history) are copied via plain `cp`, not through filter-repo.

---

## File Structure

Two new repo roots, created under `/home/adrianozdp/workspace/pfc/`:

- `gaardrail-core-new/` — filtered clone that will eventually replace `gaardrail/`. Contains `app/ cmd/ internal/ pkg/ web/`, root Go/build files, `config/config.yaml`, trimmed `Makefile`, `.github/workflows/ci.yml`, `docs/superpowers/`.
- `gaardrail-flood-test/` — filtered clone for the PFC rig. Contains `flood-test/ matlab/ docs/pfc/ brainstorming/ deploy/ docker-compose.vps.yml config/config.vps config/*.bak .github/workflows/deploy.yml`, plus a new `Makefile` and `README.md`.

Scratch clones used only as filter-repo input, deleted after the final repos are produced:
- `/tmp/claude-1000/.../scratchpad/split/core-src/`
- `/tmp/claude-1000/.../scratchpad/split/flood-src/`

## Task 1: Install git-filter-repo and stage scratch clones

**Files:** none (tooling + scratch dirs only)

**Interfaces:**
- Produces: two plain local clones of the current repo at `$SCRATCH/split/core-src` and `$SCRATCH/split/flood-src`, both with `origin` pointing at the local `gaardrail` path (not GitHub) — safe to rewrite freely.

- [ ] **Step 1: Install git-filter-repo**

Run: `pip3 install --user git-filter-repo`
Expected: install succeeds; `~/.local/bin/git-filter-repo` exists.

- [ ] **Step 2: Verify it's on PATH as a git subcommand**

Run: `git filter-repo --version`
Expected: prints a version string (e.g. `git-filter-repo <version>`). If it prints "not a git command", add `~/.local/bin` to `PATH` for the current shell (`export PATH="$HOME/.local/bin:$PATH"`) and retry.

- [ ] **Step 3: Create scratch dir and two plain clones of the live repo**

```bash
SCRATCH=/tmp/claude-1000/-home-adrianozdp-workspace-pfc-gaardrail/f8022cc3-15b4-4f98-8b07-51d9fa014572/scratchpad/split
mkdir -p "$SCRATCH"
git clone /home/adrianozdp/workspace/pfc/gaardrail "$SCRATCH/core-src"
git clone /home/adrianozdp/workspace/pfc/gaardrail "$SCRATCH/flood-src"
```

Expected: both clones report `done.` and have a full copy of the current branch (`main`).

- [ ] **Step 4: Confirm both clones are plain local clones, not linked to GitHub**

Run: `git -C "$SCRATCH/core-src" remote -v && git -C "$SCRATCH/flood-src" remote -v`
Expected: both show `origin` pointing at `/home/adrianozdp/workspace/pfc/gaardrail` (a local filesystem path), not `github.com`.

No commit — this task only sets up disposable working copies.

---

## Task 2: Filter the core clone down to controller-only history

**Files:**
- Modify (inside `$SCRATCH/core-src`, in place, via filter-repo): entire repo history rewritten to the allowlist below.

**Interfaces:**
- Consumes: `$SCRATCH/core-src` from Task 1.
- Produces: `$SCRATCH/core-src` containing only the core paths, ready for the Makefile-trim and cleanup commits in Task 3.

- [ ] **Step 1: Run filter-repo with the core path allowlist**

```bash
cd "$SCRATCH/core-src"
git filter-repo \
  --path app \
  --path cmd \
  --path internal \
  --path pkg \
  --path web \
  --path Dockerfile \
  --path go.mod \
  --path go.sum \
  --path LICENSE \
  --path README.md \
  --path openapi.yaml \
  --path docker-compose.yml \
  --path config/config.yaml \
  --path Makefile \
  --path .github/workflows/ci.yml \
  --path .gitignore \
  --path .vscode \
  --path docs/superpowers
```

Expected: filter-repo reports the rewrite completed, and removes the `origin` remote (its default safety behavior).

- [ ] **Step 2: Verify the working tree only has expected top-level entries**

Run: `cd "$SCRATCH/core-src" && ls -a`
Expected: only `app cmd internal pkg web Dockerfile go.mod go.sum LICENSE README.md openapi.yaml docker-compose.yml config Makefile .github .gitignore .vscode .git` (no `flood-test`, `matlab`, `docs/pfc`, `brainstorming`, `deploy`, `docker-compose.vps.yml`, `config/config.vps`, `config/*.bak`).

- [ ] **Step 3: Verify history was preserved, not squashed**

Run: `cd "$SCRATCH/core-src" && git log --oneline -- app/repositories/controllers/pid/pid.go | wc -l`
Expected: a number greater than 1 (multiple historical commits touched this file, matching what `git log --oneline -- app/repositories/controllers/pid/pid.go` shows in the live `gaardrail` repo).

No commit yet — Task 3 makes the follow-up content edits.

---

## Task 3: Trim the core Makefile and drop the stale openapi.yaml duplicate

**Files:**
- Modify: `$SCRATCH/core-src/Makefile`
- Delete: `$SCRATCH/core-src/docs/openapi.yaml` (only reachable if it survived the path filter — it won't, since `docs/superpowers` was the only `docs/` subpath allowlisted; this step is a no-op safety check)

**Interfaces:**
- Consumes: filtered clone from Task 2.
- Produces: final core repo content, ready to be copied into place as `gaardrail-core-new/`.

- [ ] **Step 1: Confirm docs/openapi.yaml is already gone**

Run: `test -f "$SCRATCH/core-src/docs/openapi.yaml" && echo EXISTS || echo GONE`
Expected: `GONE` (the Task 2 allowlist never included `docs/openapi.yaml`, only `docs/superpowers`). If it prints `EXISTS`, delete it: `rm "$SCRATCH/core-src/docs/openapi.yaml"`.

- [ ] **Step 2: Rewrite the Makefile to drop flood/\* targets**

Replace the full contents of `$SCRATCH/core-src/Makefile` with:

```makefile
kafka/up:
	docker compose up -d kafka

kafka/down:
	docker compose down kafka

kafka/setup:
	docker compose exec kafka /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create \
		--if-not-exists \
		--topic messages \
		--partitions 1 \
		--replication-factor 1

run:
	go run ./cmd/api

docker/build:
	docker build -t gaardrail .
```

- [ ] **Step 3: Build and test to make sure nothing load-bearing was dropped**

Run: `cd "$SCRATCH/core-src" && go build ./... && go test ./...`
Expected: both commands exit 0 with no errors (proves `go.mod`, `go.sum`, and every package the module graph needs are present).

- [ ] **Step 4: Commit the trim as a normal commit on top of the filtered history**

```bash
cd "$SCRATCH/core-src"
git add Makefile
git commit -m "Trim Makefile to core targets after flood-test split"
```

Expected: commit succeeds (working tree is otherwise clean since Step 1 was a no-op).

---

## Task 4: Filter the flood-test clone down to rig-only history

**Files:**
- Modify (inside `$SCRATCH/flood-src`, in place, via filter-repo): entire repo history rewritten to the allowlist below.

**Interfaces:**
- Consumes: `$SCRATCH/flood-src` from Task 1.
- Produces: `$SCRATCH/flood-src` containing only the flood-test/PFC paths, ready for the new Makefile/README in Task 5.

- [ ] **Step 1: Run filter-repo with the flood-test path allowlist**

```bash
cd "$SCRATCH/flood-src"
git filter-repo \
  --path flood-test \
  --path matlab \
  --path docs/pfc \
  --path brainstorming \
  --path deploy \
  --path docker-compose.vps.yml \
  --path config/config.vps \
  --path .github/workflows/deploy.yml
```

Expected: filter-repo reports the rewrite completed and removes the `origin` remote.

- [ ] **Step 2: Verify the working tree only has expected top-level entries**

Run: `cd "$SCRATCH/flood-src" && ls -a`
Expected: only `flood-test matlab docs brainstorming deploy docker-compose.vps.yml config .github .git` (no `app`, `cmd`, `internal`, `pkg`, `web`, `go.mod`, `Dockerfile`).

- [ ] **Step 3: Verify history was preserved for a known-old file**

Run: `cd "$SCRATCH/flood-src" && git log --oneline -- flood-test/docker-compose-flood.yml | wc -l`
Expected: a number greater than 1.

No commit yet — Task 5 adds the new Makefile/README and copies untracked files.

---

## Task 5: Add flood-test Makefile, README, and copy untracked files into both repos

**Files:**
- Create: `$SCRATCH/flood-src/Makefile`
- Create: `$SCRATCH/flood-src/README.md`
- Copy into `$SCRATCH/flood-src` (untracked in the live repo, no history to preserve): `docs/pfc/`, `flood-test/analysis/`, `flood-test/experiments/`, `matlab/`, `config/config.yaml.bak-pidff-t60`, `config/config.yaml.bak-revalidacao`, `flood-test/grafana/dashboards/flood.json.bak-1m`, `flood-test/grafana/dashboards/flood.json.bak-presmith`, `flood-test/prometheus.yml.bak-revalidacao`, `flood-test/scripts/closed-loop-run.sh`, `flood-test/scripts/ki-sweep.sh`, `flood-test/scripts/query-cost.sh`, `docs/superpowers/specs/2026-06-24-painel-lateral-design.md` (goes to core repo instead — see Step 3 note).

**Interfaces:**
- Consumes: filtered clone from Task 4, filtered clone from Task 3.
- Produces: both repos fully populated and ready for verification in Task 6.

- [ ] **Step 1: Write the flood-test Makefile**

Create `$SCRATCH/flood-src/Makefile`:

```makefile
flood/up:
	docker compose -f flood-test/docker-compose-flood.yml up -d

flood/down:
	docker compose -f flood-test/docker-compose-flood.yml down

flood/setup:
	./flood-test/scripts/setup-db.sh 1

flood/messages:
	./flood-test/scripts/flood.sh 10000
```

- [ ] **Step 2: Write the flood-test README**

Create `$SCRATCH/flood-src/README.md`:

```markdown
# gaardrail-flood-test

PFC (thesis) validation rig for [gaardrail](https://github.com/adrianozp/gaardrail),
a back-pressure controller for message queue consumers.

This repo holds everything needed to run closed-loop control experiments
against a running `gaardrail` instance: the flood-test infrastructure
(MySQL + Kafka + Prometheus + Grafana via Docker Compose), load-generation
and disturbance scripts, VPS deploy config, MATLAB system-identification
models, and the academic writeup (`docs/pfc/`).

It consumes the `gaardrail` controller as a published Docker image
(`adrianozdp/gaardrail`) — no controller source lives here.

## Layout

- `flood-test/` — Docker Compose stack, Grafana dashboards, experiment
  scripts and results, Prometheus scrape config.
- `matlab/` — system identification and root-locus modeling.
- `docs/pfc/` — thesis chapters and experiment writeups.
- `deploy/` — config and SQL init consumed by `docker-compose.vps.yml`
  when deploying the full rig to a VPS.
- `docker-compose.vps.yml` — full-stack deploy (gaardrail + MySQL + Kafka +
  Prometheus + Grafana + cAdvisor) for the PFC validation server.

## Running locally

```
make flood/up
make flood/setup
make flood/messages
```
```

- [ ] **Step 3: Copy untracked files from the live repo into the correct new repo**

```bash
LIVE=/home/adrianozdp/workspace/pfc/gaardrail

# Into gaardrail-flood-test
cp -r "$LIVE/docs/pfc" "$SCRATCH/flood-src/docs/pfc"
cp -r "$LIVE/flood-test/analysis" "$SCRATCH/flood-src/flood-test/analysis"
cp -r "$LIVE/flood-test/experiments" "$SCRATCH/flood-src/flood-test/experiments"
cp -r "$LIVE/matlab" "$SCRATCH/flood-src/matlab"
cp "$LIVE/config/config.yaml.bak-pidff-t60" "$SCRATCH/flood-src/config/config.yaml.bak-pidff-t60"
cp "$LIVE/config/config.yaml.bak-revalidacao" "$SCRATCH/flood-src/config/config.yaml.bak-revalidacao"
cp "$LIVE/flood-test/grafana/dashboards/flood.json.bak-1m" "$SCRATCH/flood-src/flood-test/grafana/dashboards/flood.json.bak-1m"
cp "$LIVE/flood-test/grafana/dashboards/flood.json.bak-presmith" "$SCRATCH/flood-src/flood-test/grafana/dashboards/flood.json.bak-presmith"
cp "$LIVE/flood-test/prometheus.yml.bak-revalidacao" "$SCRATCH/flood-src/flood-test/prometheus.yml.bak-revalidacao"
cp "$LIVE/flood-test/scripts/closed-loop-run.sh" "$SCRATCH/flood-src/flood-test/scripts/closed-loop-run.sh"
cp "$LIVE/flood-test/scripts/ki-sweep.sh" "$SCRATCH/flood-src/flood-test/scripts/ki-sweep.sh"
cp "$LIVE/flood-test/scripts/query-cost.sh" "$SCRATCH/flood-src/flood-test/scripts/query-cost.sh"

# Into gaardrail-core-new (this one is a software design doc, not rig material)
mkdir -p "$SCRATCH/core-src/docs/superpowers/specs"
cp "$LIVE/docs/superpowers/specs/2026-06-24-painel-lateral-design.md" \
   "$SCRATCH/core-src/docs/superpowers/specs/2026-06-24-painel-lateral-design.md"
```

Expected: all `cp` commands succeed with no output.

- [ ] **Step 4: Commit everything in the flood-test clone**

```bash
cd "$SCRATCH/flood-src"
git add -A
git commit -m "Add flood-test Makefile, README, and untracked experiment/PFC material"
```

Expected: commit succeeds, `git status` reports clean.

- [ ] **Step 5: Commit the extra spec file in the core clone**

```bash
cd "$SCRATCH/core-src"
git add docs/superpowers/specs/2026-06-24-painel-lateral-design.md
git commit -m "Add painel-lateral design spec"
```

Expected: commit succeeds, `git status` reports clean.

---

## Task 6: Verify both repos and place them at their final paths

**Files:**
- Create: `/home/adrianozdp/workspace/pfc/gaardrail-core-new/` (moved from `$SCRATCH/core-src`)
- Create: `/home/adrianozdp/workspace/pfc/gaardrail-flood-test/` (moved from `$SCRATCH/flood-src`)

**Interfaces:**
- Consumes: both finished clones from Tasks 3 and 5.
- Produces: final repos ready for user review; live `gaardrail/` untouched.

- [ ] **Step 1: Re-run build/test on the core repo as a final gate**

Run: `cd "$SCRATCH/core-src" && go build ./... && go test ./...`
Expected: both exit 0.

- [ ] **Step 2: Validate the flood-test repo's Docker Compose files parse**

```bash
cd "$SCRATCH/flood-src"
docker compose -f flood-test/docker-compose-flood.yml config >/dev/null
docker compose -f docker-compose.vps.yml config >/dev/null
```

Expected: both exit 0. (`docker-compose.vps.yml` reads `.env`-style vars like `${MYSQL_ROOT_PASSWORD}` — `docker compose config` will still succeed with warnings, not fail, since those are inline defaults-free interpolations; if it hard-fails, run with `MYSQL_ROOT_PASSWORD=x GRAFANA_ADMIN_PASSWORD=x docker compose -f docker-compose.vps.yml config >/dev/null` instead.)

- [ ] **Step 3: Spot-check no unexpected diff against the live repo's subtrees**

```bash
LIVE=/home/adrianozdp/workspace/pfc/gaardrail
diff -rq "$SCRATCH/core-src/app" "$LIVE/app"
diff -rq "$SCRATCH/flood-src/flood-test/scripts" "$LIVE/flood-test/scripts"
```

Expected: no output (identical), aside from the deliberate changes already made (Makefile, README, the moved spec file — none of which live under `app/` or `flood-test/scripts/`).

- [ ] **Step 4: Move both finished clones to their final sibling locations**

```bash
mv "$SCRATCH/core-src" /home/adrianozdp/workspace/pfc/gaardrail-core-new
mv "$SCRATCH/flood-src" /home/adrianozdp/workspace/pfc/gaardrail-flood-test
```

Expected: both directories now exist at `/home/adrianozdp/workspace/pfc/`, `$SCRATCH/split/` is empty or gone.

- [ ] **Step 5: Report final state to the user for review**

```bash
echo "=== gaardrail-core-new ===" && git -C /home/adrianozdp/workspace/pfc/gaardrail-core-new log --oneline | head -20
echo "=== gaardrail-flood-test ===" && git -C /home/adrianozdp/workspace/pfc/gaardrail-flood-test log --oneline | head -20
```

Present this output to the user along with a reminder that `gaardrail/` (the live repo) has not been modified, and ask them to review both new directories before deciding to:
(a) replace `gaardrail/` with `gaardrail-core-new/` (e.g. rename the old one aside, rename the new one into place, re-point `origin`), and
(b) create/push a GitHub remote for `gaardrail-flood-test` when ready.

No commit — this is a reporting step, not a code change.
