---
name: implement-plan
description: >-
  Use this skill when given an implementation plan file (e.g. in plans/ or specified by the user) to execute the plan step-by-step, conduct user review and feedback cycles, update DEVLOG.md following repository conventions, and stage/commit changes using conventional commits.
---

# Implement Plan Workflow

This skill guides the agent through executing an implementation plan file, managing interactive user feedback loops, updating project documentation, and creating clean conventional git commits.

## Workflow Phases

### Phase 1: Plan Analysis
1. Read the specified plan file (e.g., `plans/<plan-name>.md`) thoroughly.
2. Identify:
   - Core objectives and requirements.
   - Affected components, packages, and files.
   - List of concrete tasks and acceptance criteria.
3. Formulate an execution strategy before writing code.

### Phase 2: Execution & Automated Verification
1. Implement the planned changes step-by-step across all affected files.
2. Maintain codebase standards and preserve existing API contracts.
3. Run verification commands to ensure zero regressions:
   - Run unit tests (can be run concurrently): `go test $(go list ./... | grep -v /e2e)`
   - Run E2E tests sequentially (one at a time to avoid flakes): `go test -p 1 ./e2e/...`
   - Build binary: `go build -o lazytest .`
4. Do not proceed to user review until all tests pass cleanly.

### Phase 3: Interactive User Review Loop
1. **Present Initial Changes**:
   - Provide a concise summary of the implementation (files changed, key architectural decisions, test verification status).
   - Ask the user for review.
2. **Handle User Feedback**:
   - If the user provides feedback, requests revisions, or points out edge cases:
     a. Apply the requested modifications.
     b. Re-verify by running tests and builds.
     c. Present the updated changes to the user.
     d. Ask the user: *"Is the task complete, or would you like further changes?"*
3. **Loop Continuation**:
   - Repeat step 2 as long as the user provides additional feedback or requests modifications.
   - Proceed to Phase 4 only when the user explicitly confirms the task is complete.

### Phase 4: Update DEVLOG.md
1. Read `DEVLOG.md` to inspect current conventions for format, tone, and granularity.
2. Locate or create a section for today's date in `YYYY-MM-DD` format (e.g., `### 2026-08-02`).
3. Add concise, structured bullet points summarizing the implemented plan:
   - Format: `- **Feature Name**: Clear, high-level summary of the implemented feature or refactoring.`
   - Keep entries consistent in style, format, and length with existing log items in `DEVLOG.md`.

### Phase 5: Break Up & Conventional Git Commits
1. Inspect git status (`git status`) and modified files (`git diff --stat`).
2. Group related changes into atomic, logical commit units if the work spans multiple distinct concerns (e.g., core logic vs UI vs docs/DEVLOG).
3. Stage relevant files for each logical unit: `git add <files>`
4. Create commits using Conventional Commits format (`type(scope): message`):
   - Format: `<type>(<scope>): <short summary>`
   - Common types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`.
   - Examples:
     - `feat(ui): add toast notification system`
     - `docs(devlog): update devlog for toast notification implementation`
5. Verify `git status` is clean after committing.
