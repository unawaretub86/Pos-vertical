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

## Documents with outdated information

| Gap | Evidence | Status |
|---|---|---|
| `archive/2026-05-plan-mvp.md` reports a `cash_in` bug in the register close | `archive/2026-05-plan-mvp.md:1134` vs `internal/service/register.go:113` | **Already fixed**, the doc is what is outdated |

## To verify

Gaps that are suspected but not confirmed against the code go here. Each one is confirmed with `file:line` before moving up to the main table.
