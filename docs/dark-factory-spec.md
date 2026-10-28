# Dark Factory Spec (draft)

Autonomous development of vinyl-session-page by opencode agents on free-tier models. The human only writes high-level issues and reads reports.

## Decisions

### 1. Trust boundary: full dark
- The agent merges and deploys to prod with no human gate.
- Guardrails:
  - SQLite backup before every deploy.
  - Health check after deploy; on failure, revert the merge commit automatically.
  - Kill switch (label or repo variable) pauses the factory.
- Going sideways for a while is acceptable.

### 2. Readiness: agent triage
- The agent reads all open issues and scores them against a spec template (goal, acceptance criteria, out of scope, UI notes).
- If criteria are inferable, the agent writes them into the issue, labels it `ready-for-agent`, and proceeds.
- If criteria are ambiguous, the agent may build several prototypes and pick one.
- `needs-info` plus questions only for product decisions that belong to the human.
- The agent may create its own issues (bugs, refactors, follow-ups).
  - Label `agent-created`; max 3 open at once.
  - Human-written issues take priority.

### 3. Prototype selection: fresh-context judge with screenshots
- 2–3 prototypes maximum per ambiguous issue.
- The judge can be the same model, but with a clean context and no access to the builder's reasoning.
- The judge scores code plus Playwright screenshots against a rubric:
  - spec fit
  - consistency with existing UI
  - simplicity
  - tests pass
- The judge's reasoning goes into the PR.
- Rubric source: `docs/design-guidelines.md`, which the agent drafts from the current UI on its first run.

### 4. Verification: unit + e2e + holdout scenarios
- Pre-merge gate: `go vet`, unit tests, and Playwright e2e against a local server running on a DB snapshot.
- Unit tests only for real edge cases:
  - no tautology tests
  - no "just-in-case" tests
  - the reviewer rejects tests that don't guard real behavior
- Holdout scenarios in `scenarios/`:
  - user-journey tests written by a separate scenario agent from the issue text, before building starts
  - the builder can't edit them; a CI check fails any builder PR that touches `scenarios/`
- After deploy: smoke test on prod. Failure triggers auto-revert.

### 5. Environments: prod only
- No staging. E2E runs in CI against a local server; deploy goes straight to the droplet, then smoke test, auto-revert on failure.

### 6. Secrets and blast radius
- Agents never hold SSH keys. A GitHub Actions `deploy` job runs on push to `main` after CI is green.
- The droplet gets a dedicated non-root `deploy` user. Its SSH key has a forced command that runs only `vinyl-deploy`:
  - back up SQLite
  - swap the binary
  - restart the service
  - health check
  - roll back to the previous binary on failure
- Agent GitHub credential: a fine-grained token or GitHub App.
  - Permissions: contents, pull requests, issues, actions:read.
  - No admin and no workflow-edit rights, so the agent can't rewrite its own guardrails.
- Branch protection on `main` requires CI.
- Free-model API keys live in repo secrets, one per provider.
- The repo stays public, with an author allowlist: the agent acts only on issues by the owner or by itself; anything else gets `needs-triage` and is ignored.
- Accepted: Actions logs are public.

### 7. Reporting
- Per issue, on close, the agent comments with:
  - what was built and how
  - key decisions
  - PR link
  - screenshots
- Digest: a comment in a pinned "Factory log" issue, delivered through GitHub notifications. It covers:
  - shipped items
  - failures and rollbacks
  - pending `needs-info` questions
  - model usage
- Push alerts for urgent events only:
  - rollback
  - kill switch triggered
  - repeated failures on one issue
  - all providers exhausted
- Push channel: Telegram bot (token and chat ID in repo secrets).
- Digest: daily, only on days with activity.

### 8. Cadence: maximize free-tier use
- **Backlog mode** (workable issues exist):
  - parallel workers, each on its own issue and branch
  - runs chain back-to-back, so there is no idle time while backlog and quota remain
  - when a provider's quota runs out, switch to the next free provider
- **Idle mode** (backlog empty): event-driven. A new issue or comment from an allowlisted author wakes the factory; a cron runs as a backup.

#### 8a. Parallelism
- No fixed worker cap. One worker per non-conflicting workable issue.
- The only practical ceiling is GitHub's concurrent-job limit for free accounts (about 20).

#### 8b. Conflict avoidance
- Triage predicts which files each issue will touch; issues that overlap don't run at the same time.
- Schema migrations always run alone.
- Merge and deploy are serialized through a `deploy` concurrency group.
- Before merging, the worker rebases onto `main` and reruns CI. A conflict it can't resolve counts as a failed attempt.

#### 8c. Provider routing
- A router config lists providers and models by priority and role (builder, judge, triage).
- On 429 or quota exhaustion, the worker switches to the next provider and records the reset time.
- Quota state is kept in JSON on a `factory-state` branch.
- When all providers are exhausted, the factory sleeps until the earliest reset; this is reported in the digest, not as an alert.
- Alert only if all providers stay exhausted for more than 24h.
- 🔍 **Investigate:** reuse an existing router or fallback solution before building one (opencode provider config, OpenRouter auto-routing, LiteLLM, free-model aggregators).

#### 8d. Idle-mode triggers
- `issues` and `issue_comment` events from allowlisted authors.
- Backup cron at most hourly.
- Agent-created issues must use the GitHub App token, because `GITHUB_TOKEN` events don't trigger workflows.

### 9. Stop rules
- Max 3 attempts per issue, counted across runs.
- Worker job cap: 1h initially.
  - 📈 Re-tune from telemetry (actual run durations) once several issues have shipped.
- After the 3rd failure:
  - label `needs-human`
  - post a failure summary
  - send a Telegram alert
  - move on to the next issue
- 2 rollbacks within 24h: the factory sets the kill switch itself and sends an alert.

### 10. Self-modification
- The main agent may change the factory: prompts, rubric, router, guidelines (guardrail scope: see Q10a).
- Every `factory-change` PR must be:
  - **Documented:** what changes and which files.
  - **Reasoned:** the evidence behind it (telemetry, failures).
  - **Objected:** a separate fresh-context devil's-advocate review argues against it, and both sides are recorded in the PR.
  - **Reported specifically:** its own Telegram message plus a dedicated digest section.

#### 10a. Hard guardrails: off-limits
- The agent cannot change these; it may only propose changes as `ready-for-human` issues:
  - workflow files
  - the `scenarios/` guard
  - branch protection
  - the kill-switch check
  - the author allowlist
  - the droplet deploy script

### 11. DB migrations and rollback
- Migrations are forward-only and additive. Drops and renames happen in a later deploy, once nothing uses the column.
- Rollback swaps the binary back and keeps the current DB.
- Restoring a backup is a manual last resort, prompted by a Telegram alert.

### 12. Kill switch
- Repo variable `FACTORY_ENABLED`, checked first by every workflow.
- Flip it via the web UI or `gh variable set`; the factory flips it itself after 2 rollbacks in 24h.
- Later: Telegram `/pause` and `/resume`.

### 13. Human override
All through GitHub:
- Comment on a `needs-info` issue to answer; the comment wakes the factory.
- `priority:high` label to reorder work.
- Reopen a shipped issue with a comment to reject it; the agent treats it as a new attempt with the feedback.
- Telegram is one-way (alerts only).

### 14. Telemetry
- Placeholder: one JSONL line per run on the `factory-state` branch (issue, role, model, tokens, duration, outcome, judge scores).
- 🔍 **Investigate in a standalone grilling session**, and prefer an existing solution (e.g. opencode's built-in stats, Langfuse, OpenTelemetry-based LLM tracing).

### 15. Harness
- Build on the opencode GitHub integration for each agent role, with our own orchestration workflows for triage, fan-out, merge and deploy.
- 🔍 **Verify first:** headless runs, per-role model selection, custom agents and prompts. Fallback: the `opencode run` CLI inside Actions jobs.

### 16. Product north star
- Working assumption: `REQUIREMENTS.md`.
- 🔍 **Standalone session:** properly define the requirements and close the gaps they leave in the other areas, before the factory starts.

### 17. Test DB
- A committed minimal seed: about 20 real videos (public metadata) from the seed channels. No synthetic edge-case rows.
- Consumers:
  - holdout scenarios and Playwright e2e
  - judge screenshots
  - later, the tagging pipeline as a sample set
- Unit tests build their own fixtures.
- Schema PRs must keep the seed applying cleanly; CI fails otherwise.
- CI never touches prod data or the YouTube API.

### 18. Learning goal
- The owner wants to learn the **dark factory pattern**, not Go. The Go learning goal in `REQUIREMENTS.md` is obsolete; update it in the requirements session.
- The daily digest gets a "Factory insights" section:
  - which roles and models succeeded or failed, and why
  - where the judge and reviewer disagreed
  - which guardrails fired
  - one lesson per day
- `factory-change` PRs double as the lessons log.

### 19. Quota shared with AI tagging
- Unknown whether tagging uses the same providers or models as the factory.
- 🔍 **Investigate** together with routing (8c) and telemetry (14). Hard rule for now: the factory must not starve tagging.

## Pipeline (per issue)
1. **Triage**: check the issue's clarity, predict which files it touches, label it, and fill in acceptance criteria.
2. **Scenario agent**: write the holdout scenarios from the issue text (separate PR, auto-merged).
3. **Builder**: implement on a branch. For UI ambiguity, build 2–3 prototypes.
4. **Judge**: fresh context; pick a prototype using code, screenshots and the rubric.
5. **Reviewer**: fresh context; check spec fit, coding standards and test quality.
6. **Merge**: rebase, CI green, serialized merge.
7. **Deploy**: run `vinyl-deploy` (backup, swap, health check, smoke test, auto-rollback).
8. **Report**: issue comment, daily digest, and Telegram for urgent events.

## Follow-up sessions and investigations
- 🔍 Requirements: define them properly and close the gaps they leave in other areas (16).
- 🔍 Telemetry: standalone grilling session; prefer an existing solution (14).
- 🔍 Provider router and fallback: reuse an existing solution (8c).
- 🔍 Quota sharing with the tagging pipeline (19).
- 🔍 Capabilities of the opencode GitHub integration (15).
- 📈 Re-tune the worker time cap from telemetry (9).
