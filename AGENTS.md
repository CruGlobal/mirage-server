# Working in this repo (for coding agents)

This repository is **`mirage-server`** — Cru's HTTP redirect service. It is a
**custom [Caddy](https://caddyserver.com) distribution** written in Go: the
`mirage` binary is Caddy plus this repo's own modules, which resolve a redirect
for an incoming hostname out of **DynamoDB** and issue it, and which obtain and
store **Let's Encrypt certificates** for every hostname pointed at it (also in
DynamoDB, so certificates survive task replacement and are shared across tasks).

It runs as a long-lived container on **AWS ECS** behind a **network** load
balancer (TCP 80 + 443 — TLS terminates in the container, not the LB, because
the container owns the ACME flow). Health check: `GET /.ping`.

It is the replacement for the legacy `redirector` app. Retiring `redirector`
is a separate piece of work — nothing in this repo drives it.

## Layout & language

```
.
├── AGENTS.md / CLAUDE.md
├── Caddyfile           # the shipped server config (adapter: caddyfile)
├── Dockerfile          # 2-stage caddy:<ver>-builder-alpine -> caddy:<ver>-alpine
├── build.sh            # what CI runs to build the image (docker buildx)
├── Taskfile.yml        # go-task: build / fmt / lint / test / run / dynamodb:*
├── .tool-versions      # asdf: caddy, golang, golangci-lint, pre-commit
├── compose.yml         # DynamoDB Local, for tests and local runs
├── .github/workflows/test.yml                # PR checks: Build / Format / Lint + Test
├── .github/workflows/conventional-commits.yml # PR check: Validate PR Title
├── .github/workflows/pipeline-v2.yml         # nightly build + release-candidate deploy
├── .github/workflows/build-deploy-ecs.yml    # parked v1 workflow
├── cmd/mirage/         # the binary (Caddy's cmd wiring)
├── internal/
│   ├── app/            # Caddyfile directives / adapter glue
│   ├── mirage/         # the HTTP handler module
│   ├── redirect/       # redirect model: type, status, rewrite rules
│   ├── storage/        # DynamoDB: config items, cert storage, distributed lock
│   ├── cache/          # in-process cache in front of DynamoDB
│   └── permission/     # cert-issuance permission checks
└── miragetest/         # test helpers
```

Go (version pinned in `.tool-versions`), Caddy v2 modules, AWS SDK v2
(DynamoDB).

**The Caddy version lives in `.tool-versions`, not the Dockerfile.** `build.sh`
greps it out and passes it as `--build-arg CADDY_VERSION`; the Dockerfile's
`ARG CADDY_VERSION=0` default is a placeholder that no real build uses. Bump
Caddy by editing `.tool-versions` — and bump the Caddy Go modules in `go.mod`
to match, or the custom build and the base image disagree.

## The loop

| Command | What it does |
| --- | --- |
| `task build` | `go build -o mirage ./cmd/mirage` |
| `task fmt` | `golangci-lint fmt` |
| `task lint` | `golangci-lint run --fix` |
| `task test` | `go test -cover ./...` (needs DynamoDB Local) |
| `task dynamodb:start` / `dynamodb:stop` | DynamoDB Local via `compose.yml` |
| `task run` | run the server locally against `assets/config/Caddyfile` |
| `task mirage:validate` | validate the shipped `Caddyfile` |
| `./build.sh` | build the container image exactly the way CI does |

`.github/workflows/test.yml` reports two checks: **`Build / Format / Lint`** and
**`Test`**. Together with **`Validate PR Title`** those are the repo's
**required status checks** in the `pipeline-v2-default-branch` ruleset. The
ruleset is repo configuration in `cru-terraform`, not a file in this repo, so
renaming a job silently drops the gate — change both together (ask DevOps if you
can't edit rulesets). Dependabot patch/minor and security PRs auto-merge once
they are green (`.github/workflows/dependabot-auto-merge.yml`).

`task fmt` / `task lint` rewrite files, and CI fails on any resulting diff — run
them before pushing. `pre-commit` runs both plus whitespace hooks.

## How this app ships

This repo is on **pipeline v2: build once, then promote the artifact**. One
environment-agnostic image is built from `main`, deployed to
**release-candidate** (the stage surface), and — if it is good — promoted
byte-for-byte (by digest) to **production**. The authoritative reference is
[`docs/pipeline-v2.md`](https://github.com/CruGlobal/cru-deploy/blob/main/docs/pipeline-v2.md)
in `CruGlobal/cru-deploy`.

There is **no `staging` branch, no `On Staging` label, and no merge-bot** in this
flow. If you find those referenced anywhere, the reference is stale.

1. **Work on a branch** off `main` and open a **Pull Request** back to `main`.
   The repo is **squash-only with auto-merge enabled**: turn auto-merge on
   (`gh pr merge --auto --squash`) and GitHub merges it when the required checks
   go green.
2. **Write the PR title as a Conventional Commit** (`feat: …`,
   `fix(redirect): …`) — it becomes the squash commit subject, and
   `Validate PR Title` enforces it.
3. **Builds do not run on push.** `.github/workflows/pipeline-v2.yml` runs on a
   **nightly cron at 05:00 UTC** (`cron: '0 5 * * *'` — midnight EST / 1am EDT,
   one unstaggered slot shared by the whole v2 fleet) and on manual
   **`workflow_dispatch`**. Merging to `main` does not deploy anything by
   itself. Scheduled dispatch is best-effort, so a nightly that starts a few
   minutes late is expected, not a failure.
4. **A build produces a candidate.** It pushes `candidate-<yyyy-mm-dd>-<n>` and
   `sha-<gitsha>` to the app's ECR repo
   (`056154071827.dkr.ecr.us-east-1.amazonaws.com/mirage-server`), then
   dispatches a deploy of that candidate to **release-candidate**. A night with
   no new commits is a true end-to-end no-op: the build reuses the already-built
   sha, and the deploy skips a digest release-candidate is already running.
5. **Promotion and rollback live in
   [`cru-deploy`](https://github.com/CruGlobal/cru-deploy)**, not here — the
   **Promote (v2)** and **Rollback (v2)** `workflow_dispatch` Actions. Promote
   re-deploys the exact digest already running on release-candidate and stamps it
   `release-<yyyy-mm-dd>-<n>`; it first checks that the person dispatching it has
   **push permission on this repo**. `release-*` tags are permanent, so any past
   release stays rollback-able. **Deploy Candidate (v2)** (`force: true`) is the
   force-redeploy escape hatch.
6. **Slack.** Every deploy, promote, and rollback — and every failure, as an
   `:x:` message — posts to **`#devops-notifications`**, the channel in this
   app's `CruApplicationInfo` row (`SlackChannel`), with a
   `deploys.cru.org/changelog` link for what changed. Deploys run in cru-deploy's
   Actions tab, so Slack (or that tab) is where you see results; a red cru-deploy
   run does not turn this repo red.
7. **The image is environment-agnostic — never bake environment-specific values
   into it.** The same bytes run on release-candidate and production; every
   `MIRAGE_*` and `DYNAMODB_*` value arrives as a task-definition environment
   variable from Terraform. That includes the ACME directory: stage points at
   Let's Encrypt **staging** and production at the real one, which is a
   parameter, not a build flag. The one build-identity exception is the pair at
   the *end* of the `Dockerfile`:

   ```dockerfile
   ARG VERSION="dev"
   ENV DD_VERSION=${VERSION}
   ```

   The build passes `--build-arg VERSION=<yyyy-mm-dd>-<n>`, so Datadog's
   `version` is the build's identity in every environment. Keep those two lines
   last and don't add environment config beside them.
8. **The v1 workflow is parked, not removed.**
   `.github/workflows/build-deploy-ecs.yml` still exists but its push triggers
   are gone — it is `workflow_dispatch`-only, an escape hatch back to the v1
   build+deploy path. Don't use it for normal work, and don't add push triggers
   to it.

Watch a run with `gh run watch`, or the Actions tab here (build) and in
`cru-deploy` (deploy/promote/rollback).

> Pilot note: `pipeline-v2.yml` pins the reusable workflow and actions to the
> `@pipeline-v2` branch of `CruGlobal/.github` and passes
> `workflow-ref: pipeline-v2` so both match. Those references get re-pinned to
> `@v2` when the pipeline is released — change them together or not at all.

### What an ECS deploy actually does

Worth knowing, because it explains where task-definition changes come from:

- **Terraform owns the task-definition template.** A deploy reads the **latest
  revision of the task-definition family** (Terraform's template), composes a new
  revision from it changing **only** the app container's `image` (to the resolved
  digest) and refreshing its runtime secrets from SSM
  (`/ecs/mirage-server/<prod|stage>/`), registers that, and updates every
  matching service. Sidecars pass through untouched.
- **Consequence:** a Terraform change to cpu/memory, a `MIRAGE_*` parameter, log
  config, or a sidecar lands on the *next* deploy. If nothing else changed, pick
  it up with **Deploy Candidate (v2)** and `force: true` — a plain nightly would
  no-op because the digest is unchanged.
- **Deploys are digest-pinned.** Tags are resolved to a `sha256:…` digest first;
  handing the deploy a tag is an error by design.
- **No database migrations here.** This app has no relational database and no
  migrations at all; Terraform declares `database_migrations = { enabled = false }`
  with **no `path`**, which is what makes promote report rollback safety as
  **`safe — no database migrations`** rather than leaving it `unclassified`.
  DynamoDB tables are Terraform resources, not migrations. If you ever add a
  schema-ful datastore, that declaration has to change with it.
- **Certificates are state, and they live in DynamoDB, not in the image.** A
  deploy replaces tasks; the new tasks read the existing certificates back out of
  the certs table. Nothing about a deploy re-issues certificates, and a rollback
  does not invalidate them.

## Feature flags

The pipeline runs a feature-flag service and the `aws/ecs/app` module injects
`CRU_FLAGS_URL` into release-candidate and production — but **mirage-server has
no flag client wired up**, so the variable is present and unread. Wire one the
day there is a flag to read, and never write your own flag system (no env-var
toggle, no DynamoDB config item, no hand-rolled fetch of `CRU_FLAGS_URL`). Off
is the safe default: `CRU_FLAGS_URL` is unset locally, in CI and in tests, and
unknown flags are `false`. Detail: the [pipeline guide's Feature flags
section](https://github.com/CruGlobal/cru-deploy/blob/main/docs/pipeline-v2.md#feature-flags);
flags per environment are on <https://deploys.cru.org>.

## Infrastructure & secrets

- **Provisioning** (the ECS cluster/service, task role, network load balancer,
  DynamoDB config + certificates tables, Route 53 records, SSM parameters,
  deploy permissions) lives in
  [`cru-terraform`](https://github.com/CruGlobal/cru-terraform) under
  `applications/mirage-server/` via the `aws/ecs/app` module. Don't hand-write
  cloud infrastructure.
- **Redirects themselves are Terraform too.** A redirect is a
  `applications/mirage/redirect` module call that writes the DynamoDB config
  item and the DNS record together (see
  `applications/mirage-server/stage/redirects.tf` for the shape). Adding a
  redirect is a `cru-terraform` PR, never a hand-edited DynamoDB item.
- **Runtime metadata the pipeline reads** (provider, type, Slack channel,
  migrations) is the app's `CruApplicationInfo` row, written by that module.
  Read it at
  `https://deploys.cru.org/info?project=mirage-server&environment=production`.
- **Secrets** live in SSM Parameter Store under `/ecs/mirage-server/<env>/` and
  are attached to the container by the deploy — never commit secrets. Use
  `cru application impersonate -e staging -- <command>` to run against a real
  environment's values.
- **BUILD-time secrets** (if ever needed) are Actions secrets on this repo named
  `BUILD_<NAME>`; the build exports only `BUILD_*` into the build environment
  with the prefix stripped.

## Leftovers you can ignore

- A stale **`staging` branch** still exists on the remote — it was the v1
  merge-bot's deploy target. `main` is the only live branch; nothing builds from
  `staging` any more.
- The **`On Staging` label** is likewise dead. It drove the v1 merge-bot, which
  this repo no longer runs (`.github/merge-bot.yml` is gone).
- `.github/workflows/build-deploy-ecs.yml` is the **parked** v1 workflow. It is
  intentionally present and intentionally dispatch-only.
- `ARG CADDY_VERSION=0` in the `Dockerfile` is a placeholder default, not a
  version anyone ships. The real value comes from `.tool-versions` via
  `build.sh`.

## If you're not sure what to do

- **Keep changes small and on a branch.** Open a PR; don't push to `main`.
- **Don't invent infrastructure.** New AWS resources, SSM parameters, DNS
  records, or redirects are a `cru-terraform` change — say so rather than
  configuring things by hand.
- **Never paste secrets** into files. Use env vars and the Cru CLI for real
  values.
- **Be careful around ACME and certificate storage.** This service holds live
  Let's Encrypt certificates for real Cru hostnames in DynamoDB, and Let's
  Encrypt rate-limits issuance hard. Exercise cert paths against the **staging**
  ACME directory and DynamoDB Local, never against production.
- **Confirm before anything outward-facing or hard to undo** — pushing, opening
  PRs, dispatching a promote or rollback, deleting things.
