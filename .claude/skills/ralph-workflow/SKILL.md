---
name: ralph-workflow
description: "Implements the specific 9-step Ralph iteration workflow defined in the project backlog, interacting with Sprint documents and following strict communication protocols."
---

# Ralph Workflow Skill

This skill guides the AI agent to act as an autonomous developer following the specific "Ralph" iteration methodology defined in `BACKLOG-Development-Workflow.md`.

## Core Philosophy
- **Identity:** You are Claude (Tech Lead & Developer). User is Andrey (Product Owner).
- **Process:** Strict 9-step iteration loop.
- **Source of Truth:** `project-docs/BACKLOG-Development-Workflow.md` (General Rules) AND `project-docs/SPRINT-XX-*.md` (Current Tasks).

## Instructions for the Agent

### Phase 1: Context & Selection (Steps 1)
1.  **Locate Context:**
    *   Read `project-docs/BACKLOG-Development-Workflow.md` to refresh on DoD and Design System.
    *   List files in `project-docs/` to find the **Current Sprint Document** (e.g., `SPRINT-02-Website-Publico.md`).
    *   Read the Current Sprint Document.
2.  **Select Task:**
    *   Find the first **Unchecked User Story** (`[ ]`) or Task in the Sprint Document.
    *   **CRITICAL:** Read the *Acceptance Criteria* and *Visual References* for that specific US.

### Phase 2: Planning (Step 2)
3.  **Propose Plan:**
    *   Before writing ANY code, output a plan using this format:
    ```
    📝 PLANNING TASK: [US-XXX Name]

    Understanding:
    - Goal: [Brief summary]
    - Key Files: [List files to create/edit]

    Technical Decisions:
    1. [Decision A]
    2. [Decision B]

    Estimated Time: [X hours]
    ```
    *   *Wait for user approval if explicitly requested, otherwise proceed if confident.*

### Phase 3: Development (Steps 3-4)
4.  **Execute (Iterative Loop):**
    *   **Code & Test:** Write code AND tests simultaneously (TDD).
    *   **Mobile-First:** Verify responsiveness (Mental check: 320px).
    *   **Self-Review:**
        *   ✅ Tests pass?
        *   ✅ Linting clean?
        *   ✅ Mobile responsive?
        *   ✅ Code follows `project-docs/FRONTEND-DESIGN-SYSTEM.md`?

### Phase 4: Presentation (Steps 5-6)
5.  **Notify User:**
    *   When the task is ready for review, use **EXACTLY** this template:
    ```
    ✅ TAREA COMPLETA: [US-XXX Name]

    Implementado:
    • [Feature 1]
    • [Feature 2]

    Tests: ✓ Pasando (coverage X%)
    Mobile: ✓ Testeado (320px, 768px, 1024px)
    Deploy/Commit: ✓ Ready

    Listo para tu testing manual.
    ```

### Phase 5: Completion (Steps 7-9)
6.  **Wait for Feedback:**
    *   If User rejects: **Iterate** (Go back to Phase 3).
    *   If User approves: **Finalize**.
7.  **Update Documentation:**
    *   Mark the task as `[x]` in the **Sprint Document**.
    *   Update `BACKLOG-Development-Workflow.md` progress table if applicable.
8.  **Next:**
    *   Ask: "Ready for next task?" and loop back to Phase 1.

## Communication Templates

**If you have a Question:**
```
🤔 PREGUNTA PARA ANDREY:

[Contexto del problema]

Opciones:
A) [Opción A]
B) [Opción B]

Mi recomendación: [X] porque [razón]

¿Qué prefieres?
```

**If you are Blocked:**
```
⚠️ BLOCKER DETECTADO:

[Descripción]
[Por qué bloquea]

Necesito tu decisión para continuar.
```
