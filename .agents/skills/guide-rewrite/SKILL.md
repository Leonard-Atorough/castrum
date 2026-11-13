---
name: guide-rewrite
description: "Plan, write, improve, or review Castrum tutorials and how-to guides. Use for reader-centered documentation structure, API-grounded workflows, examples, and focused validation."
argument-hint: "[plan|implement|review] <reader-goal-or-guide-path> [owning-package-or-folder]"
user-invocable: true
---

# Guide Rewrite

Use this workflow for Castrum tutorials and how-to guides. It also applies when planning or reviewing their structure. Read and follow the shared [documentation editorial contract](../../../docs/CONTRIBUTING.md) before acting. Ground every behavioral claim in the public API and relevant working examples or tests.

## Modes and scope

Choose a mode from the request. Default to `implement` for a request to write or update documentation and to `plan` for a request about structure or approach.

- `plan`: read-only; propose an outcome, structure, ownership, and checks.
- `implement`: make the requested or agreed change and validate it.
- `review`: read-only; report actionable findings and residual risk.

Identify the reader's goal, intended audience, target version, affected guide or area, owning public API, and closest runnable example or test. If the guide exists, read it first and preserve valid user-authored material. Inspect only the related material needed to understand the task; a cross-cutting workflow may involve more than one package.

Keep changes within the requested scope. Do not move or split pages, change released snapshots, or alter examples or Go docs unless requested or necessary to correct an affected link or claim. If a necessary change would materially broaden scope, explain it and ask before proceeding. Follow the repository's versioning policy: development documentation lives under `docs/unreleased`, while released snapshots describe their matching tagged API.

## Procedure

### 1. Establish the contract

Read the current guide, owning public API, and the closest relevant example or test. Determine:

- The outcome, prerequisites, setup order, and observable result.
- Relevant ownership, lifecycle, scheduling, caching, validation, and error timing.
- Which example can be reused and whether it matches the target version.
- Which claims are stale, which constraints matter to this task, and where related details already belong.

Distinguish verified facts from assumptions. State a concise implementation hypothesis and a focused validation check before editing.

### 2. Plan

Return the proposed page purpose and heading hierarchy, reader workflow, API facts and limitations, existing material and examples to reuse, ownership and links, and focused validation. For a structural plan, map affected existing material to its proposed home and identify link and version compatibility. Do not edit files in this mode.

### 3. Implement

Apply the editorial contract to the requested change. Preserve useful user-authored material and use ASCII by default.

After editing, check the page end to end for a reader's ability to complete the task. Verify API names, signatures, links and anchors, example version, error handling, and claims against the implementation. Run the smallest relevant executable check for the owning package or example when available, followed by existing documentation checks and `git diff --check`. State explicitly when a snippet, interactive step, link, or command was not verified. Do not claim that a package test proves independent Markdown snippets compile.

### 4. Review

Do not edit. Report findings first, ordered by severity: incorrect behavior or examples; missing constraints, ownership, or failure timing; broken links or version mismatches; then confusing reader flow, duplicated ownership, and prose. Cite the affected location and practical consequence. End with checks performed and residual risk; say clearly if there are no findings.

## Completion

A rewrite is complete when its primary purpose and reader outcome are clear, its claims match the target API, necessary constraints and examples are accurate, links resolve to appropriate versions, and focused checks pass or their limitations are reported. Do a final prose and navigation check without expanding scope.
