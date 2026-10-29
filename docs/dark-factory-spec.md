# Dark Factory Spec (draft)

Autonomous development of vinyl-session-page by opencode agents on free-tier models. The human only writes high-level issues and reads reports.

## Decisions

### 1. Trust boundary: full dark, reached in phases
- End state: the agent merges and deploys to prod with no human gate.
- **Phase 0, shadow mode:** the factory runs the whole pipeline but stops at an open PR; the owner merges. It runs for at least 20 PRs in total; the owner reviews each and records whether they agree with the judge and reviewer.
  - Why 20: with 0 disagreements in n PRs, the 95% upper bound on the true disagreement rate is about 3/n (rule of three). 2 PRs bound it at 78%, 10 at 26%, 20 at about 14%.
- **Phase 1, full dark:** auto-merge turns on **per tier** (1a), low first, then medium. A tier is promoted when:
  - at least 10 shadow PRs of that tier
  - owner agreement with judge and reviewer ≥ 90%
  - 0 Important findings the gates missed (the owner found them, not the reviewer)
- **Sampling after promotion:** the owner reads 1 in 5 auto-merged PRs of a newly promoted tier until 20 have been read. Disagreements become replay eval cases (10b).
- **Demotion:** a tier drops back to shadow mode when its revert rate exceeds 10% over its last 10 merges, or for all tiers when the error budget is spent (9a). Only the owner promotes again.
- Guardrails:
  - SQLite backup before every deploy.
  - Health check after deploy; on failure, revert the merge commit automatically.
  - Kill switch (12) pauses the factory.
- Going sideways for a while is acceptable.

#### 1a. Risk tiers
- The tier is set by a classifier from the change (paths touched, diff content), not by the author or the agent's own claim.
- Tiers:
  - **Low:** docs, CSS, copy, additive UI. Auto-merge once all checks are green.
  - **Medium:** ordinary features. Auto-merge in phase 1.
  - **High:** schema migrations, ingest and YouTube quota use, auth, deploy-related code. The owner approves; an agent approval never counts as the owner's.
- Off-limits paths (10a) are never merged by the factory, whatever the tier.
- The classifier rules and path lists are off-limits (10a).
- **Enforcement (personal repo, no org features):**
  - GitHub App approvals count toward required reviews, and push restrictions are org-only, so tiers are not enforced by required approvals.
  - Instead, a required check `tier-gate`, pinned to the merger App, carries the tier decision. It goes green for promoted tiers (1). For high tier, a `pull_request_review` workflow sets it green only when the reviewer is the owner and the approval is on the current head SHA; a new commit resets it.
  - "Require branches to be up to date before merging" is on, so every merge has run CI on the latest `main`.

### 2. Readiness: agent triage
- The agent reads all open issues and scores them against a spec template (goal, acceptance criteria, out of scope, UI notes).
- If criteria are inferable, the agent writes them into the issue, labels it `ready-for-agent`, and proceeds.
  - In phase 0, and always for high tier, inferred criteria wait for the owner's 👍 reaction before building starts.
  - Every failed attempt is labeled `spec-defect` (the criteria were wrong) or `impl-defect` (the build was wrong) in telemetry (14).
- **Frozen target:** acceptance criteria and holdout scenarios are fixed before building starts. Their hash goes into the PR; changing either one counts as a new attempt.
- If criteria are ambiguous, the agent may build several prototypes and pick one.
- `needs-info` plus questions only for product decisions that belong to the human.
- The agent may create its own issues (bugs, refactors, follow-ups).
  - Label `agent-created`; max 3 open at once, max 3 created per day.
  - Depth limit 1: work on an agent-created issue can't create another agent-created issue.
  - Overflow: when a cap is full, the item goes into the digest (7) instead of being dropped.
  - Each body ends with a `source:` block (review finding, telemetry, seed data). Text quoted from third-party content stays marked as data, so it doesn't become trusted just because the agent is an allowlisted author (6).
  - Agent-created issues can't be `factory-change` issues or touch off-limits paths (10a).
  - Human-written issues take priority.

### 3. Prototype selection: fresh-context judge with screenshots
- 2–3 prototypes maximum per ambiguous issue, only if the remaining daily request budget (8c) covers them; otherwise one.
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
- **Testing focus:** holdout scenarios and Playwright e2e are the main safety net. Unit tests only for real edge cases:
  - no tautology tests
  - no "just-in-case" tests
  - the reviewer rejects tests that don't guard real behavior
- **Test integrity:** a CI guard fails any builder PR that deletes an e2e test or adds `test.skip`, `test.fixme` or `test.only`, unless the acceptance criteria remove that behavior. The reviewer checks for weakened assertions.
- Holdout scenarios in a private repo, `vinyl-session-page-scenarios`:
  - user-journey tests written by a separate scenario agent from the issue text, before building starts
  - the scenario agent runs on a different model family than the builder
  - **red first:** a new scenario must fail on current `main`; a scenario that passes immediately is rejected
- **Holdout isolation (private repo):**
  - Only the scenario agent's app (write) and the scenario runner's app (read) are installed on the scenarios repo. The builder's app is not, so it can neither read nor edit scenarios.
  - The satisfaction gate runs in the scenarios repo's own Actions, triggered by `repository_dispatch` with the PR head SHA. Its logs stay private; it reports only the score and pass/fail as a check on the public PR.
  - Builder instructions: treat the acceptance criteria as the only target.
  - The reviewer flags any builder transcript that references the scenarios repo; a hit fails the attempt.
  - Constraint: GitHub Free includes 2000 Actions minutes per month for private repos, each job rounded up to a full minute. See the minutes budget (8e).
- **Hard checks vs satisfaction:**
  - Hard requirements are deterministic Playwright assertions: one run, one retry for flakes.
  - LLM-judged satisfaction is only for subjective outcomes. Each such scenario runs N times (start with 3).
  - Satisfaction is the share of satisfying trajectories across all judged runs, not per scenario. Merge needs satisfaction ≥ 0.9 over at least 10 trajectories, and no scenario at 0/N.
  - Why: per scenario, N=3 can only score 0, 0.33, 0.67 or 1, so a 0.9 threshold would mean 3/3. LLM judges flip on identical input often enough that 3/3 would fail good PRs.
- **Existing scenarios:** must stay passing. A hard check that fails, then passes on its retry, counts as passing and is logged as flaky; a scenario flaky 3 times becomes a `ready-for-human` issue.
- **YouTube twin:** CI runs against a fake YouTube Data API server built from recorded responses, including quota errors and 403s, so ingest features can be validated without the real API.
- After deploy: smoke test on prod plus a read-only subset of the holdout scenarios. Failure triggers auto-revert.
- Synthetic monitoring:
  - hourly smoke test from the public repo (free minutes); it is the SLO probe (9a)
  - the holdout subset runs once a day from the scenarios repo (private minutes)

#### 4a. Code review
- **Self-review before the PR:** after tests, build and lint pass, the builder runs a fresh-context subagent on its own diff and fixes what it finds.
- **Reviewer = hunter + verifier:**
  - a bug hunter reads the diff against the whole repo and the acceptance criteria
  - a verifier from a different model family than the hunter confirms each finding; only confirmed findings are posted
  - findings cover bugs and spec fit only; style, cleanup and diagnostics are dropped
  - each finding cites a file and line as evidence
- **Severity:** each finding is tagged:
  - **Important:** blocks the merge until fixed or resolved with a reason
  - **Pre-existing:** a bug the PR didn't introduce; becomes an `agent-created` issue (within the cap in 2) and doesn't block
  - nits are not posted
- **Convergence:**
  - at most 3 review-and-fix rounds per attempt; the 4th counts as a failed attempt (9)
  - after the first round, only the new diff is re-reviewed
  - a posted finding is never repeated; findings name the class of bug, not only one line
- **`REVIEW.md`** is the reviewer's repo guide (off-limits, 10a):
  - what failure looks like here
  - paths to skip
  - the evidence rule
- **Fix at the source:** when the reviewer flags the same class of bug repeatedly, the factory adds a lint rule or updates `AGENTS.md`. Changes to `REVIEW.md` go through a `ready-for-human` issue.

#### 4b. PR template
- `.github/pull_request_template.md` (off-limits, 10a) with these sections:
  - **Issue:** `Closes #`
  - **What changed**
  - **Testing:** commands run, e2e and scenario results, screenshots
  - **Risk:** tier (1a) and why
  - **How to revert:** `git revert <sha>`, plus migration notes if any
  - **Provenance:** criteria hash, model ID per role, satisfaction, review rounds
- A CI check fails the PR if any section is empty.

### 5. Environments: prod only
- No staging. E2E runs in CI against a local server; deploy goes straight to the droplet, then smoke test, auto-revert on failure.

### 6. Secrets and blast radius
- Agents never hold SSH keys. A GitHub Actions `deploy` job runs on push to `main` after CI is green.
- The SSH key lives only in a `production` GitHub environment restricted to `main`. The deploy job runs no agent-editable script before it uses the key.
- The droplet gets a dedicated non-root `deploy` user. Its SSH key has a forced command that runs only `vinyl-deploy`:
  - verify the binary's build attestation
  - back up SQLite (11)
  - swap the binary
  - restart the service
  - health check
  - roll back to the previous binary on failure
- CI attests the built binary (GitHub artifact attestations); `vinyl-deploy` refuses unattested binaries.
- Agent identity: **GitHub Apps only** (short-lived installation tokens), one per role:
  - triage: issues
  - builder: contents, pull requests (branch protection keeps it off `main`); not installed on the scenarios repo
  - scenario agent: contents on the scenarios repo only
  - scenario runner: read on the scenarios repo, checks on this repo
  - merger workflow: merges after required checks pass, posts `tier-gate` (1a); the builder never merges its own PR
  - none get admin, workflow-edit, or variables rights, so agents can't rewrite their own guardrails or the kill switch.
- **Merge audit:** the builder needs `contents: write` to push branches, so it could technically merge. A workflow on push to `main` checks that the merge was done by the merger App; otherwise it reverts, sends a Telegram alert, and the watchdog turns the factory off (12).
- Branch protection on `main`:
  - required checks: CI, test-integrity guard, PR template check, satisfaction gate, reviewer, `tier-gate`
  - each required check is pinned to its expected source app, so other workflows can't spoof it
  - "Require branches to be up to date before merging" on
  - "Do not allow bypassing the above settings" on
- Repo setting "Allow GitHub Actions to create and approve pull requests" is off.
- Free-model API keys live in repo secrets, one per provider, and stay out of the agent's shell environment (requests go through the router).
- Provider allowlist in the router: only providers with known data policies; opt out of training where offered. Never put secrets in model context. Log the exact model ID per run.
  - Accepted: some free tiers use prompts for training and human review with no opt-out (e.g. the unpaid Gemini API). The repo is public and secrets never enter model context, so this is allowed; the allowlist records each provider's policy.
- Untrusted input:
  - The repo stays public, with an author allowlist: the agent acts only on issues by the owner or by itself; anything else gets `needs-triage` and is ignored.
  - Prompts mark third-party text (YouTube titles and descriptions in the seed, dependency docs, web pages) as data, never instructions.
  - Runner egress is allowlisted to the providers, GitHub, and package registries with StepSecurity Harden-Runner in block mode (free for public repos).
    - Accepted: Harden-Runner needs a paid license for private repos, so the scenarios repo has no egress block. It holds only a judge provider key and check-write rights.
- **Workflow injection** (PromptPwnd):
  - never interpolate `${{ github.event.* }}` inside `run:`; pass it through `env:`
  - no `pull_request_target` workflow checks out PR code
  - `zizmor` runs in CI on all workflow files
- Accepted: Actions logs are public. A log scan fails the run if a secret pattern appears.
- **Cost:** all factory work runs in the public repo, where Actions minutes are free. Only the satisfaction gate and the daily holdout run use private minutes (8e). The Actions spending limit stays at $0, so GitHub blocks runs at the quota instead of billing.

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
  - merge not done by the merger App (6)
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
- Models: a mix of free models from several providers, configured as opencode providers.
- Buy $10 of OpenRouter credits once: it lifts the `:free` daily cap from 50 to 1000 requests. At 50/day the factory can't finish one issue a day.
- A router config lists providers and models by priority and role (builder, judge, reviewer, scenario, triage).
- **Capacity:** free tiers cap requests per minute and per day (e.g. OpenRouter `:free`: 20/min, and 50/day with under $10 of purchased credits). One agent run takes dozens to hundreds of requests, so budgets count requests as well as tokens.
  - Phase 0 measures requests per issue per role.
  - The digest reports capacity as issues per day.
  - The parallelism cap (8a) never exceeds what the per-minute limits allow.
- The router enforces model-family separation: judge, reviewer and scenario agent never share a family with the builder of the same issue, and the review verifier never shares one with the bug hunter (4a).
- A new model ID for any role is scored against the replay eval set (10b) before it takes real work.
- On 429 or quota exhaustion, the worker switches to the next provider and records the reset time.
- Token budgets per issue and per day, per role. Hitting the issue budget counts as a failed attempt; the daily budget reserves a share for tagging (19).
- Quota state is kept in JSON on a `factory-state` branch. Only a state workflow writes it, through a `factory-state` concurrency group; workers send it events and never push to the branch.
- When all providers are exhausted, the factory sleeps until the earliest reset; this is reported in the digest, not as an alert.
- Alert only if all providers stay exhausted for more than 24h.
- 🔍 **Investigate:** reuse an existing router or fallback solution before building one (opencode provider config, OpenRouter auto-routing, LiteLLM, free-model aggregators).

#### 8d. Idle-mode triggers
- `issues` and `issue_comment` events from allowlisted authors.
- Backup cron at most hourly.
- Agent-created issues must use the GitHub App token, because `GITHUB_TOKEN` events don't trigger workflows.

#### 8e. Private Actions minutes budget
- 2000 minutes per month on the scenarios repo. Rough use:
  - satisfaction gate: about 2 runs × 5 min per issue
  - holdout subset after each deploy: about 3 min
  - daily holdout run: about 90 min per month
  - that leaves room for about 140 issues per month
- Go and Playwright browsers are cached to cut setup time.
- Usage is tracked in `factory-state` and reported in the digest.
- Above 80% used: drop the daily holdout run, then cut N (4) to 1.
- At the cap, the satisfaction gate stays pending and nothing merges until the monthly reset.
- No self-hosted runner on the droplet: it would run builder-written code on the prod box.

### 9. Stop rules
- Max 3 attempts per issue, counted across runs.
  - Infra failures don't count: 429s, provider outages, runner timeouts before the agent starts. They retry after the provider's reset time.
- Worker job cap: 1h initially.
  - 📈 Re-tune from telemetry (actual run durations) once several issues have shipped.
- Each worker writes a progress file for its issue on the branch; a retry resumes from it instead of starting cold. The file records the criteria hash (2); a retry ignores a file whose hash doesn't match the frozen one.
- After the 3rd failure:
  - label `needs-human`
  - post a failure summary
  - send a Telegram alert
  - move on to the next issue

#### 9a. SLO and error budget
- SLO, measured over a rolling 4 weeks:
  - availability ≥ 99% on the hourly smoke probe (4), so about 6 failed probes out of 672
  - holdout satisfaction on prod ≥ 0.9
- Budget spent: the factory stops feature work and only takes fixes until the budget recovers, and sends an alert.
- An incident that uses more than 20% of the budget needs a postmortem in its `factory-change` PR or issue, and becomes a replay eval case (10b).
- Hard backstop: 2 rollbacks within 24h trip the kill switch (12).
- **Revert fast path:** after a failed health check, `vinyl-deploy` has already swapped the binary back, so prod recovers first. The merger App then opens the revert PR as an exact `git revert <sha>`. The reviewer, satisfaction and `tier-gate` Apps recognize it and pass right away; CI still runs.

### 10. Self-modification
- The main agent may change the factory: builder and triage prompts, router, guidelines other than the rubric (guardrail scope: see 10a).
- Every `factory-change` PR must be:
  - **Documented:** what changes and which files.
  - **Reasoned:** the evidence behind it (telemetry, failures).
  - **Objected:** a separate fresh-context devil's-advocate review argues against it, and both sides are recorded in the PR.
  - **Scored:** run against the replay eval set (10b); a regression blocks the merge.
  - **Reported specifically:** its own Telegram message plus a dedicated digest section.

#### 10a. Hard guardrails: off-limits
- The agent cannot change these; it may only propose changes as `ready-for-human` issues:
  - workflow files, and every script, Makefile target or config they execute
  - the scenarios repo, its workflows and app installations
  - the test-integrity guard and the holdout-isolation instructions
  - judge, reviewer and scenario-agent prompts, and `REVIEW.md` (4a)
  - the risk-tier classifier and its path lists (1a)
  - the replay eval set (10b)
  - the rubric (`docs/design-guidelines.md`) and the satisfaction threshold
  - the PR template (4b)
  - SLO and error budget policy
  - branch protection and required-check sources
  - the kill-switch check and watchdog
  - the author allowlist and provider allowlist
  - the droplet deploy script
- Enforced by CODEOWNERS on these paths plus a CI check that fails any agent PR touching them.

#### 10b. Replay eval set
- Incidents become evals: every rollback, reopened issue and `needs-human` failure adds a case, confirmed by the owner.
- Seeded from the phase 0 issues; grows with every incident and every owner disagreement during sampling (1).
- Scores every `factory-change` PR (10) and every new model ID (8c). The eval runs in a workflow, so agents can't edit results.
- Also runs weekly in the public repo, because free models can change behind the same ID. A score drop demotes that model ID in the router.

### 11. DB migrations and rollback
- Migrations are forward-only and additive. Drops and renames happen in a later deploy, once nothing uses the column.
- Rollback swaps the binary back and keeps the current DB.
- Backups:
  - **Before each deploy:** `sqlite3 app.db ".backup /var/backups/pre-<sha>.db"`. This uses SQLite's Online Backup API, which gives a consistent snapshot of a live database; a plain file copy can be corrupt or miss WAL data.
  - **Off-site, continuous:** Litestream streams the WAL to a free-tier S3-compatible bucket (Cloudflare R2 or Backblaze B2, about 10 GB free). Daily snapshots, retention `336h` (14 days).
- A weekly CI job runs `litestream restore` and runs the smoke test against the restored DB.
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
- Per review finding: severity, and how the builder responded (fixed, said wrong, said intentional, didn't matter).
- Derived:
  - change failure rate and failed-deploy recovery time (DORA)
  - owner/judge agreement (1)
  - review rounds per attempt (4a)
  - reviewer false-alarm rate: the reviewer is rerun on merged PRs that were never reverted, and every finding there counts as a false alarm
  - escape rate per gate: the share of changes that passed the gate and were later reverted or reopened; the leakiest gate gets fixed first
  - `spec-defect` vs `impl-defect` share of failures (2)
  - private Actions minutes used (8e)
- The log is append-only and written by a workflow, not by agents, so agents can't edit their own history.
- 🔍 **Investigate in a standalone grilling session** which store to use, and prefer an existing solution (e.g. opencode's built-in stats, Langfuse, OpenTelemetry-based LLM tracing).

### 15. Harness
- Build on the opencode GitHub integration for each agent role, with our own orchestration workflows for triage, fan-out, merge and deploy.
- 🔍 **Verify first:** headless runs, per-role model selection, custom agents and prompts. Fallback: the `opencode run` CLI inside Actions jobs.

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
- Dependabot PRs:
  - patch and minor updates of existing deps, and Actions SHA bumps: low tier, auto-merged once checks are green
  - major updates: `ready-for-human`

### 21. Provenance
- The merger adds commit trailers to every factory merge: `Factory-Issue`, `Criteria-Hash`, `Tier`, `Models` (one ID per role), `Satisfaction`.
- Together with the build attestation (6), any merge can be traced back to its spec, models and gate results from git alone.

## Pipeline (per issue)
1. **Triage**: check the issue's clarity, predict which files it touches, label it, and fill in acceptance criteria.
2. **Scenario agent**: write the holdout scenarios from the issue text in the private scenarios repo, prove they fail on `main` (auto-merged there). Criteria and scenarios are frozen.
3. **Builder**: implement on a branch, with no access to the scenarios repo. For UI ambiguity, build 2–3 prototypes. Self-review in a fresh context before opening the PR (4a).
4. **Judge**: fresh context, different model family; pick a prototype using code, screenshots and the rubric.
5. **Reviewer**: fresh context, different model family; bug hunter plus verifier check spec fit, bugs, test quality and holdout isolation. At most 3 review-and-fix rounds (4a).
6. **Satisfaction gate**: in the scenarios repo, run hard checks once and judged scenarios N times; merge needs all hard checks green and satisfaction ≥ threshold (4).
7. **Merge** (promoted tiers only; others stop at an open PR): classify the risk tier (1a) and post `tier-gate`; high waits for the owner's approval on the head SHA. Rebase, required checks green, serialized merge by the merger workflow with provenance trailers (21).
8. **Deploy**: run `vinyl-deploy` (verify attestation, backup, swap, health check, smoke test and prod scenarios, auto-rollback).
9. **Report**: issue comment, daily digest, and Telegram for urgent events.

## Follow-up sessions and investigations
- 🔍 Requirements: define them properly and close the gaps they leave in other areas (16).
- 🔍 Telemetry store: standalone grilling session; prefer an existing solution (14).
- 🔍 Provider router and fallback: reuse an existing solution (8c).
- 🔍 Quota sharing with the tagging pipeline (19).
- 🔍 Capabilities of the opencode GitHub integration (15).
- 📈 Re-tune the worker time cap, parallelism cap, satisfaction threshold and N from telemetry (4, 8a, 9).

## References
- StrongDM Software Factory (holdout scenarios, satisfaction, digital twins): https://factory.strongdm.ai/
- Anthropic, Solving code review at Anthropic (risk tiers, verified findings, round cap, incidents as evals): https://claude.dev/blog/solving-code-review-at-anthropic/
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
- OpenRouter free-model rate limits: https://openrouter.ai/docs/api/reference/limits
- Gemini API terms (unpaid services): https://ai.google.dev/gemini-api/terms
- Anthropic, Effective harnesses for long-running agents (don't remove or edit tests): https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
- Augment, Dark factory (escape rate per gate, provenance, reviewing agent merges): https://www.augmentcode.com/guides/dark-factory-autonomous-software-delivery
- Rule of three (zero-failure upper bound ≈ 3/n): https://jmgamble.substack.com/p/the-rule-of-3-in-critical-appraisal
- τ-bench (pass^k): https://arxiv.org/abs/2406.12045
- LLM-as-a-judge verdict flips on repeated runs: https://arxiv.org/abs/2512.16041
- GitHub Actions billing (per-job rounding, $0 spending limit, public repos free): https://docs.github.com/billing/managing-billing-for-your-products/managing-billing-for-github-actions
- GitHub branch protection rules (push restrictions org-only): https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule
- GitHub App reviews count toward required approvals: https://github.com/orgs/community/discussions/181487
- StepSecurity Harden-Runner: https://docs.stepsecurity.io/harden-runner/enable-runtime-security/github-hosted-runners
- SQLite Online Backup API: https://www.sqlite.org/backup.html
- Litestream: https://litestream.io/how-it-works/
