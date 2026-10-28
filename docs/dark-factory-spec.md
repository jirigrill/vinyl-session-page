# Dark Factory Spec (draft)

Autonomous development of vinyl-session-page by opencode agents on free-tier models. The human only writes high-level issues and reads reports.

## Decisions

### 1. Trust boundary: full dark, reached in phases
- End state: the agent merges and deploys to prod with no human gate.
- **Phase 0, shadow mode:** the factory runs the whole pipeline but stops at an open PR; the owner merges. For the first N issues (start with 10), record whether the owner agrees with the judge and reviewer.
- **Phase 1, full dark:** auto-merge turns on once owner/judge agreement and the telemetry (14) look acceptable.
- Guardrails:
  - SQLite backup before every deploy.
  - Health check after deploy; on failure, revert the merge commit automatically.
  - Kill switch (12) pauses the factory.
- Going sideways for a while is acceptable.

### 2. Readiness: agent triage
- The agent reads all open issues and scores them against a spec template (goal, acceptance criteria, out of scope, UI notes).
- If criteria are inferable, the agent writes them into the issue, labels it `ready-for-agent`, and proceeds.
- **Frozen target:** acceptance criteria and holdout scenarios are fixed before building starts. Their hash goes into the PR; changing either one counts as a new attempt.
- If criteria are ambiguous, the agent may build several prototypes and pick one.
- `needs-info` plus questions only for product decisions that belong to the human.
- The agent may create its own issues (bugs, refactors, follow-ups).
  - Label `agent-created`; max 3 open at once, max 3 created per day.
  - Agent-created issues can't be `factory-change` issues or touch off-limits paths (10a).
  - Human-written issues take priority.

### 3. Prototype selection: fresh-context judge with screenshots
- 2–3 prototypes maximum per ambiguous issue.
- The judge runs on a **different model family** than the builder, with a clean context and no access to the builder's reasoning. LLM judges favor output from their own family.
- The judge scores code plus Playwright screenshots against a rubric:
  - spec fit
  - consistency with existing UI
  - simplicity
  - tests pass
- The judge's reasoning goes into the PR.
- Rubric source: `docs/design-guidelines.md`. The agent drafts it from the current UI on its first run; the owner approves it once, then it is frozen (10a).

### 4. Verification: unit + e2e + holdout scenarios
- Pre-merge gate: `go vet`, unit tests, and Playwright e2e against a local server running on a DB snapshot.
- Unit tests only for real edge cases:
  - no tautology tests
  - no "just-in-case" tests
  - the reviewer rejects tests that don't guard real behavior
- Holdout scenarios in `scenarios/`:
  - user-journey tests written by a separate scenario agent from the issue text, before building starts
  - the scenario agent runs on a different model family than the builder
  - **red first:** a new scenario must fail on current `main`; a scenario that passes immediately is rejected
  - the builder can't edit them; a CI check fails any builder PR that touches `scenarios/`
- **Holdout isolation (no separate repo):**
  - Builder instructions: never read, list, grep, or open anything under `scenarios/`; never infer scenario content from CI logs; treat the acceptance criteria as the only target.
  - Defense in depth: the builder job checks out the repo with sparse checkout that excludes `scenarios/`, and opencode's permission config denies the path where supported.
  - The reviewer flags any builder transcript that references `scenarios/`; a hit fails the attempt.
  - Accepted risk: the repo is public, so the isolation is behavioral, not absolute.
- **Satisfaction instead of pass/fail:** the judge runs each scenario N times (start with 3) and reports the share of runs that satisfy the user. Merge needs satisfaction ≥ threshold (start at 0.9). All existing scenarios must stay passing.
- **YouTube twin:** CI runs against a fake YouTube Data API server built from recorded responses, including quota errors and 403s, so ingest features can be validated without the real API.
- After deploy: smoke test on prod plus a read-only subset of the holdout scenarios. Failure triggers auto-revert. The same subset runs hourly as synthetic monitoring.

### 5. Environments: prod only
- No staging. E2E runs in CI against a local server; deploy goes straight to the droplet, then smoke test, auto-revert on failure.

### 6. Secrets and blast radius
- Agents never hold SSH keys. A GitHub Actions `deploy` job runs on push to `main` after CI is green.
- The SSH key lives only in a `production` GitHub environment restricted to `main`. The deploy job runs no agent-editable script before it uses the key.
- The droplet gets a dedicated non-root `deploy` user. Its SSH key has a forced command that runs only `vinyl-deploy`:
  - verify the binary's build attestation
  - back up SQLite
  - swap the binary
  - restart the service
  - health check
  - roll back to the previous binary on failure
- CI attests the built binary (GitHub artifact attestations); `vinyl-deploy` refuses unattested binaries.
- Agent identity: **GitHub Apps only** (short-lived installation tokens), one per role:
  - triage: issues
  - builder: contents, pull requests (branch protection keeps it off `main`)
  - merger workflow: merges after required checks pass; the builder never merges its own PR
  - none get admin, workflow-edit, or variables rights, so agents can't rewrite their own guardrails or the kill switch.
- Branch protection on `main`:
  - required checks: CI, scenario guard, satisfaction gate, reviewer
  - each required check is pinned to its expected source app, so other workflows can't spoof it
  - "Do not allow bypassing the above settings" on
- Free-model API keys live in repo secrets, one per provider, and stay out of the agent's shell environment (requests go through the router).
- Provider allowlist in the router: only providers with known data policies; opt out of training where offered. Never put secrets in model context. Log the exact model ID per run.
- Untrusted input:
  - The repo stays public, with an author allowlist: the agent acts only on issues by the owner or by itself; anything else gets `needs-triage` and is ignored.
  - Prompts mark third-party text (YouTube titles and descriptions in the seed, dependency docs, web pages) as data, never instructions.
  - Runner egress is allowlisted to the providers, GitHub, and package registries.
- Accepted: Actions logs are public. A log scan fails the run if a secret pattern appears.

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
  - model usage and token spend
  - error budget remaining (9a)
- Push alerts for urgent events only:
  - rollback
  - kill switch triggered
  - repeated failures on one issue
  - all providers exhausted
  - error budget spent
- Push channel: Telegram bot (token and chat ID in repo secrets).
- Digest: daily, only on days with activity.

### 8. Cadence: maximize free-tier use
- **Backlog mode** (workable issues exist):
  - parallel workers, each on its own issue and branch
  - runs chain back-to-back, so there is no idle time while backlog and quota remain
  - when a provider's quota runs out, switch to the next free provider
- **Idle mode** (backlog empty): event-driven. A new issue or comment from an allowlisted author wakes the factory; a cron runs as a backup.

#### 8a. Parallelism
- Start with a cap of 2 workers. Raise it from telemetry (conflict rate, failed attempts), not from GitHub's concurrent-job limit (about 20 on free accounts).

#### 8b. Conflict avoidance
- Triage predicts which files each issue will touch; issues that overlap don't run at the same time.
- Schema migrations always run alone.
- Merge and deploy are serialized through a `deploy` concurrency group.
- Before merging, the merger rebases onto `main` and reruns CI. A conflict that can't be resolved counts as a failed attempt.

#### 8c. Provider routing
- A router config lists providers and models by priority and role (builder, judge, reviewer, scenario, triage).
- The router enforces model-family separation: judge, reviewer and scenario agent never share a family with the builder of the same issue.
- On 429 or quota exhaustion, the worker switches to the next provider and records the reset time.
- Token budgets per issue and per day, per role. Hitting the issue budget counts as a failed attempt; the daily budget reserves a share for tagging (19).
- Quota state is kept in JSON on a `factory-state` branch; writes go through a `factory-state` concurrency group.
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
- Each worker writes a progress file for its issue on the branch; a retry resumes from it instead of starting cold.
- After the 3rd failure:
  - label `needs-human`
  - post a failure summary
  - send a Telegram alert
  - move on to the next issue

#### 9a. SLO and error budget
- SLO: site availability plus holdout satisfaction on prod (4), measured over 4 weeks.
- Budget spent: the factory stops feature work and only takes fixes until the budget recovers, and sends an alert.
- Hard backstop: 2 rollbacks within 24h trip the kill switch (12).

### 10. Self-modification
- The main agent may change the factory: builder and triage prompts, router, guidelines other than the rubric (guardrail scope: see 10a).
- Every `factory-change` PR must be:
  - **Documented:** what changes and which files.
  - **Reasoned:** the evidence behind it (telemetry, failures).
  - **Objected:** a separate fresh-context devil's-advocate review argues against it, and both sides are recorded in the PR.
  - **Reported specifically:** its own Telegram message plus a dedicated digest section.

#### 10a. Hard guardrails: off-limits
- The agent cannot change these; it may only propose changes as `ready-for-human` issues:
  - workflow files, and every script, Makefile target or config they execute
  - the `scenarios/` guard and the holdout-isolation instructions
  - judge, reviewer and scenario-agent prompts
  - the rubric (`docs/design-guidelines.md`) and the satisfaction threshold
  - SLO and error budget policy
  - branch protection and required-check sources
  - the kill-switch check and watchdog
  - the author allowlist and provider allowlist
  - the droplet deploy script
- Enforced by CODEOWNERS on these paths plus a CI check that fails any agent PR touching them.

### 11. DB migrations and rollback
- Migrations are forward-only and additive. Drops and renames happen in a later deploy, once nothing uses the column.
- Rollback swaps the binary back and keeps the current DB.
- Backups are copied off the droplet, with a retention window (start with 14 days).
- A weekly CI job restores the latest backup and runs the smoke test against it.
- Restoring a backup in prod is a manual last resort, prompted by a Telegram alert.

### 12. Kill switch
- Repo variable `FACTORY_ENABLED`, checked first by every workflow and again before each pipeline step.
- A separate watchdog workflow, with its own GitHub App, can only set it to off: after 2 rollbacks in 24h or when the error budget is spent.
- Only the owner turns it back on (web UI or `gh variable set`).
- Later: Telegram `/pause` and `/resume`.

### 13. Human override
All through GitHub:
- Comment on a `needs-info` issue to answer; the comment wakes the factory.
- `priority:high` label to reorder work.
- Reopen a shipped issue with a comment to reject it; the agent treats it as a new attempt with the feedback.
- Telegram is one-way (alerts only).

### 14. Telemetry
- **Prerequisite for phase 0**, not a follow-up.
- Minimum per run: issue, role, model ID, provider, tokens, tool calls, duration, outcome, judge scores, satisfaction.
- Derived: change failure rate and failed-deploy recovery time (DORA), owner/judge agreement (1).
- The log is append-only and written by a workflow, not by agents, so agents can't edit their own history.
- 🔍 **Investigate in a standalone grilling session** which store to use, and prefer an existing solution (e.g. opencode's built-in stats, Langfuse, OpenTelemetry-based LLM tracing).

### 15. Harness
- Build on the opencode GitHub integration for each agent role, with our own orchestration workflows for triage, fan-out, merge and deploy.
- 🔍 **Verify first:** headless runs, per-role model selection, custom agents and prompts, per-path read permissions (4). Fallback: the `opencode run` CLI inside Actions jobs.

### 16. Product north star
- Working assumption: `REQUIREMENTS.md`.
- 🔍 **Standalone session:** properly define the requirements and close the gaps they leave in the other areas, before the factory starts.

### 17. Test DB
- A committed minimal seed: about 20 real videos (public metadata) from the seed channels. No synthetic edge-case rows.
- Seed text is third-party content and is treated as untrusted (6).
- Consumers:
  - holdout scenarios and Playwright e2e
  - judge screenshots
  - later, the tagging pipeline as a sample set
- Unit tests build their own fixtures.
- Schema PRs must keep the seed applying cleanly; CI fails otherwise.
- CI never touches prod data or the YouTube API; it uses the YouTube twin (4).

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
- 🔍 **Investigate** together with routing (8c) and telemetry (14). Hard rule for now: the factory must not starve tagging; the daily token budget reserves its share.

### 20. Supply chain
- New Go modules, npm packages or GitHub Actions need a `ready-for-human` issue; the agent can't add them itself.
- Actions pinned to full commit SHAs.
- `govulncheck` in CI; Dependabot on.

## Pipeline (per issue)
1. **Triage**: check the issue's clarity, predict which files it touches, label it, and fill in acceptance criteria.
2. **Scenario agent**: write the holdout scenarios from the issue text, prove they fail on `main` (separate PR, auto-merged). Criteria and scenarios are frozen.
3. **Builder**: implement on a branch, without reading `scenarios/`. For UI ambiguity, build 2–3 prototypes.
4. **Judge**: fresh context, different model family; pick a prototype using code, screenshots and the rubric.
5. **Reviewer**: fresh context, different model family; check spec fit, coding standards, test quality and holdout isolation.
6. **Satisfaction gate**: run the scenarios N times; merge needs satisfaction ≥ threshold.
7. **Merge** (phase 1 only; phase 0 stops at an open PR): rebase, required checks green, serialized merge by the merger workflow.
8. **Deploy**: run `vinyl-deploy` (verify attestation, backup, swap, health check, smoke test and prod scenarios, auto-rollback).
9. **Report**: issue comment, daily digest, and Telegram for urgent events.

## Follow-up sessions and investigations
- 🔍 Requirements: define them properly and close the gaps they leave in other areas (16).
- 🔍 Telemetry store: standalone grilling session; prefer an existing solution (14).
- 🔍 Provider router and fallback: reuse an existing solution (8c).
- 🔍 Quota sharing with the tagging pipeline (19).
- 🔍 Capabilities of the opencode GitHub integration, including per-path read permissions (15).
- 📈 Re-tune the worker time cap, parallelism cap, satisfaction threshold and N from telemetry (4, 8a, 9).

## References
- StrongDM Software Factory (holdout scenarios, satisfaction, digital twins): https://factory.strongdm.ai/
- Dan Shapiro, The Five Levels: https://www.danshapiro.com/blog/2026/01/the-five-levels-from-spicy-autocomplete-to-the-software-factory/
- OWASP Top 10 for Agentic Applications 2026: https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026
- GitHub Actions secure use: https://docs.github.com/en/actions/reference/security/secure-use
- GitHub protected branches (check sources, no bypass): https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
- GitHub artifact attestations: https://docs.github.com/en/actions/concepts/security/artifact-attestations
- Aikido, PromptPwnd: https://aikido.dev/blog/promptpwnd-github-actions-ai-agents
- Self-Preference Bias in LLM-as-a-Judge: https://arxiv.org/abs/2410.21819
- ImpossibleBench (reward hacking on tests): https://en.papernotes.org/ICLR2026/llm_safety/impossiblebench_measuring_llms_propensity_of_exploiting_test_cases/
- Google SRE example error budget policy: https://sre.google/workbook/error-budget-policy/
- OpenRouter data logging and training: https://openrouter.ai/docs/guides/privacy/logging
