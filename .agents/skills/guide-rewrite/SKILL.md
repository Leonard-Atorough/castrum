---
name: guide-rewrite
description: "Plan, implement, or review a repository guide grounded in its owning API. Use for documentation rewrites, guide structure planning, API-accurate examples, and focused post-edit validation."
argument-hint: "[plan|implement|review] <guide-path> [owning-package-or-folder]"
user-invocable: true
---

# Guide Rewrite

Use this skill for a user-facing guide whose behavior is controlled by a local package, folder, example, or test suite.

## Modes

Choose one mode from the request. If no mode is given, use `implement` when the user asks to write or update the guide, and use `plan` when the user asks only for structure or approach.

- `plan`: read-only analysis; do not edit files.
- `implement`: apply the agreed guide rewrite and validate it.
- `review`: read-only API and prose review; do not edit files.

## Required Inputs

Identify these anchors before acting:

- The current guide path.
- The owning package or folder containing the public API.
- A neighboring example, test, or related guide when one exists.

If the guide path is known, read it first and preserve user edits. Do not map the whole repository.

## Procedure

### 1. Establish the contract

Read the current guide and the owning public API. Then inspect only the closest test, runnable example, or related guide needed to answer these questions:

- What behavior does the API actually support?
- What setup order, validation, ownership, caching, or lifecycle rules matter to readers?
- Which examples can be copied or adapted from working code?
- Which claims in the current guide are stale or stronger than the implementation?

State one concise implementation hypothesis and one focused validation check before editing.

### 2. Plan mode

Return a compact plan containing:

- The proposed heading hierarchy.
- The reader workflow from setup to common use.
- API facts and constraints the guide must explain.
- Examples or related guides to reuse.
- Known limitations that should be stated explicitly.
- The focused validation to run after implementation.

Do not modify files in this mode.

### 3. Implement mode

Rewrite only the target guide unless a linked example or documentation reference must be corrected for accuracy.

Use these writing rules:

- Speak naturally and instructively to the reader.
- Lead with the supported mental model and the shortest successful path.
- Organize sections by the reader's decisions and workflow, not by source-file order.
- Prefer working API examples from the repository; keep snippets focused.
- Distinguish setup-time validation from lazy or runtime resolution.
- State current limitations instead of implying unsupported features.
- Link to neighboring guides for concepts already explained elsewhere.
- Preserve valid user-authored material and avoid unrelated reformatting.
- Use ASCII by default and avoid filler, placeholders, and speculative future promises.

After the edit, run the focused executable check when available, then documentation checks. Typical checks are:

```text
go test ./<owning-package>
git diff --check
```

Also inspect headings, API identifiers, links, and the guide end to end for a prose sniff test. Fix local documentation defects and rerun the same focused checks.

### 4. Review mode

Do not edit. Review the guide against the owning API and report findings first, ordered by severity:

- Incorrect or unsupported API claims.
- Examples that cannot work with the current public API.
- Missing lifecycle, validation, error, ownership, or setup-order rules.
- Broken or misleading links.
- Remaining prose or structure problems.

End with test gaps or residual documentation risk. If there are no findings, say so clearly.

## Multi-agent Dispatch Contract

When dispatching multiple agents, give each the same target guide and owning package, and assign one mode per agent:

1. `plan`: proposes the structure and factual contract.
2. `implement`: receives the approved plan and edits only the target guide.
3. `review`: checks the implementation against the owning API and reports findings.

The implementation agent must not broaden scope because another agent found an unrelated issue. The review agent must not silently edit; return findings for a follow-up implementation pass.

## Completion Criteria

A guide rewrite is complete when:

- Its structure matches the approved reader workflow.
- Its claims match the owning implementation.
- Examples use current public names and signatures.
- Current limitations and failure timing are clear.
- The focused package check passes, or the reason it could not run is reported.
- `git diff --check` passes.
- A final end-to-end prose sniff finds no obvious stale claims, placeholders, or awkward transitions.
