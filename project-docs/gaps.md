> **Type:** current · **Rules over:** what is missing for production · **Updated:** 2026-10-01

This document lists **verified** limitations and debt of the current product; what will be built in the future lives in [roadmap.md](roadmap.md), which is a different thing.

## Open debt (not roadmap, it is what was left half-done)

| US / Sprint | What | Since |
|---|---|---|
| US-034 | Factus: full sandbox validation | SPRINT-07 — requires external access |
| SPRINT-05 | Mobile Responsive, marked pending although the style base is already in use | SPRINT-05 |
| US-042c, US-043, US-045 | Sales speed, customer history in the POS, WhatsApp catalog | SPRINT-09 |
| US-053 | Register-close summary by email — there is uncommitted work in `internal/mailer/` | SPRINT-12 |
| US-054 | POS access over local WiFi | SPRINT-12, reassigned to SPRINT-13 |
| US-055 to US-058 | Full restaurant module | SPRINT-13 |
| US-059 to US-062 | Local assistant | SPRINT-14 |

## Pending PO validations (Andrey, manual)

Code and tests are done for these; only a human check on a real browser or device is missing. Parked by the PO on 2026-10-20 until he has time. Remove a row when the check is done.

| US | What to check | Where it is described |
|---|---|---|
| US-077 | In DevTools set `pos_token` to `x`, click any view: lands on `/login`. Wrong password on `/login`: shows "credenciales inválidas" without reloading. | `SPRINT-18-Estabilizacion-Produccion.md` (US-077), contract §4 |
| US-076 | AC-7: as a cashier, inspect the network on `/setup` in a browser and confirm `owner_email_app_password` is absent; as admin, save Settings and confirm the stored email password is kept. | `SPRINT-18-Estabilizacion-Produccion.md` (US-076) |
| US-078 | Confirm 3 product choices made by the Tech Lead: customers deleted logically (`active = 0`), delete answers 409 when the balance is above 0, credit sales (`fiado`) to inactive customers are rejected. Open the app on an existing database and check that migration 19 runs. | `roadmap.md` R-56 and decision log 2026-10-13 |
| US-062 | M-1..M-9 on a real phone: visual, theme, 320 px, keyboard. | `workflow.md` SPRINT-14 row |
| US-054 | Open `/login` from the compiled binary and from a phone or tablet over WiFi (AC-16). | `roadmap.md` R-55 |
| US-074 | Responsive 320 px on a phone or emulator. | `workflow.md` SPRINT-17 row |
| US-034 | Factus full sandbox validation (needs external access). | Open debt table above |

## Documents with outdated information

| Gap | Evidence | Status |
|---|---|---|
| `archive/2026-05-plan-mvp.md` reports a `cash_in` bug in the register close | `archive/2026-05-plan-mvp.md:1134` vs `internal/service/register.go:113` | **Already fixed**, the doc is what is outdated |

## To verify

Gaps that are suspected but not confirmed against the code go here. Each one is confirmed with `file:line` before moving up to the main table.
