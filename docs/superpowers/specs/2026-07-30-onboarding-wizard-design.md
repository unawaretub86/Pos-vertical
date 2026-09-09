# Onboarding Wizard + Branding — Design

## Problem

`pos.exe` boots straight to `/login`. There is no way to create the first real
admin user or configure the business — the only account is a hardcoded DB
seed (`admin` / `admin`, user id 1, `internal/database/migrate.go:259-261`).
Any further users or business config changes require already being logged in
as that admin. There is also no product branding: the login page shows a
placeholder `<div>` with the text "POS" (`frontend/app/login/page.tsx:52-66`)
instead of a real logo, and the favicon is the unmodified Next.js default.

## Goals

- First launch walks the owner through setting real admin credentials and
  minimal business info, then drops them into the app logged in.
- Give the product a real identity ("POS Chévere") with a designed icon mark,
  used on the login page and as the favicon.

## Non-goals

- Multi-tenant / multi-business support. This stays a single-tenant desktop
  app — there is no `tenant_id` concept anywhere in the schema and this
  design does not introduce one.
- Re-running onboarding after it's been completed once.
- Hardening the API against a user who bypasses the frontend and calls
  `/api/auth/login` with the default `admin`/`admin` credentials directly
  before completing onboarding. See "Accepted risk" below.

## Decisions

### 1. First-run detection

Add a `setup_completed BOOLEAN NOT NULL DEFAULT false` column to the existing
single-row `business_config` table (already the source of truth for business
settings, `internal/handler/setup.go`). A new public (no-auth) endpoint,
`GET /api/setup/status`, returns `{"completed": bool}`.

Rejected alternatives:
- Heuristic based on whether the seeded admin's password is still literally
  `"admin"` — already used for a *different* purpose (`force_password_change`
  in `internal/handler/auth.go:52`) and fragile: it breaks if an admin ever
  resets their password back to `"admin"`, and conflates "default password"
  with "onboarding done".
- Inferring from whether `business_config` fields are empty — ambiguous if a
  business later clears its own config.

### 2. Admin bootstrap

The wizard does not create a new user row. It updates the existing seeded
admin (id 1) in place: sets `full_name`, `username`, and `password` (hashed)
via the same validation/hashing path already used by `createUser` /
`updateUser` in `internal/handler/user.go` (extracted into a shared
service call rather than duplicated).

Rejected alternative: create a brand-new admin user and deactivate the seed.
More moving parts (two admin rows to reconcile, need to decide what happens
to id 1 in the interim) for no behavioral benefit over reusing id 1.

### 3. Business fields captured by the wizard

Minimal only: `business_name`, `nit`. Currency is hardcoded to `"COP"` server
-side (this app's Colombian electronic-invoicing integration, `internal/factelect`,
makes COP the only realistic value — asking is friction with no payoff).
Everything else already in `BusinessConfig` (address, ticket header/footer,
logo) stays optional, editable later at `/setup` exactly as it is today.

### 4. Flow / UX shape

Two-step wizard with a stepper (progress dots), not a single scrolling form:
- **Step 1 — "Tu cuenta":** full name, username, password, confirm password.
- **Step 2 — "Tu negocio":** business name, NIT.
- Final submit button: "Empezar a vender".

Step state lives in React state only (no draft persistence) — losing
progress on a page refresh is an acceptable tradeoff for a two-field-per-step
form; adding persistence would be scope creep for the size of this form.

### 5. Branding

Product name: **"POS Chévere"**. Icon mark: a price-tag silhouette with a
checkmark inside (SVG), rendered in a rounded `accent-dim` box, matching the
app's existing dark theme tokens (`--canvas: #0D0D0D`, `--accent: #F5A623`,
`frontend/app/globals.css`). Wordmark: "POS **Chévere**" with "Chévere" in
the accent color.

This is a `<Logo>` React component (SVG inline, no external asset files),
used in two places:
- `frontend/app/login/page.tsx`, replacing the placeholder text div
  (lines 52-66).
- The new onboarding page header.

The same icon is exported as a static favicon, replacing the untouched
default Next.js `frontend/app/favicon.ico`.

Mockups reviewed and approved via the visual-companion session
(`.superpowers/brainstorm/2084-1785439243/content/logo-concepts.html`,
`wizard-layout.html`).

## Architecture

### Backend (Go)

- **Migration:** add `setup_completed` column to `business_config`
  (`internal/database/migrate.go`), default `false`.
- **`GET /api/setup/status`** (public, no auth middleware): returns
  `{"completed": bool}`. Added alongside the existing `/auth/login` public
  route in `internal/handler/handler.go`.
- **`POST /api/onboarding`** (public, no auth middleware):
  - Body: `{full_name, username, password, business_name, nit}`.
  - Guard: if `setup_completed` is already `true`, return `409 Conflict` and
    do nothing else (idempotency / re-run protection).
  - Validates and hashes the password using the same helper `createUser`
    uses today.
  - Updates user id 1 (`full_name`, `username`, `password_hash`).
  - Updates `business_config` (`name`, `nit`, `currency = "COP"`,
    `setup_completed = true`).
  - Issues a JWT the same way `login` does (`appmw.GenerateToken`) so the
    frontend can go straight from "submit" to "logged in", no second login
    step.

### Frontend (Next.js)

- **`frontend/app/onboarding/page.tsx`** (new): two-step stepper UI per the
  approved mockup (dot indicators, `--accent` highlight on the active step).
  Calls `POST /api/onboarding` on final submit, stores the returned token via
  the existing `setAuth` helper (`frontend/lib/auth.ts`), redirects to the
  dashboard.
- **`frontend/app/login/page.tsx`**: on mount, calls `GET /api/setup/status`;
  if `completed === false`, `router.replace('/onboarding')` before rendering
  the login form. Replace the placeholder logo div with `<Logo>`.
- **`frontend/app/onboarding/page.tsx`**: on mount, calls the same status
  endpoint; if `completed === true`, `router.replace('/login')` (blocks
  re-running the wizard by navigating to the URL directly).
- **New shared component** `frontend/components/Logo.tsx`: the SVG icon +
  wordmark, used by both pages above.
- **Favicon:** regenerate `frontend/app/favicon.ico` from the same mark.

## Accepted risk

The seeded `admin`/`admin` credentials keep working against
`POST /api/auth/login` even while `setup_completed` is `false` — enforcement
of the onboarding gate lives in the frontend redirect, not the backend. This
app runs on `localhost` only, single-tenant, not internet-exposed, so someone
would need local shell access to the machine to exploit this — at which
point they already have full access to the SQLite file directly. Closing
this gap server-side (e.g. blocking `/auth/login` entirely until setup is
complete) was considered and rejected as unnecessary complexity for the
actual threat model.

## Testing

- **Backend:** unit tests for `POST /api/onboarding` — happy path, guard
  rejects when `setup_completed = true`, validation errors (weak password,
  duplicate username if applicable).
- **Frontend:** manual smoke test via the `run` skill after implementation —
  fresh DB boot lands on `/onboarding`, complete the wizard, confirm redirect
  to dashboard logged in, confirm `/login` and `/onboarding` redirect
  correctly into each other post-setup, confirm the new logo renders on
  `/login` and the favicon updated.
