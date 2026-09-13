# Roster balance CLI

## Calendar ingestion

For v0, the CLI stays dumb about calendar providers and accepts normalized input through a single adapter layer. The first supported import format should be ICS exports, which lets Outlook and other corporate calendars work without committing to Graph or OAuth integration. Direct Outlook sync can come later only if the operational need justifies the extra scope.
In the future, a Starlark-based ingestion layer could map raw calendar events into roster concepts like availability, absences, or team-specific exceptions while keeping provider connectors in Go.

## Identity and event effects

External inputs are resolved to a stable internal `MemberID` before they enter the domain model. Ingress adapters may start with an email address, calendar identity, or provider-specific user ID, but policies do not resolve identities themselves.

The event flow is:

```text
external input
  -> identity resolution
  -> domain event with canonical member ID
  -> track record
  -> Starlark event rule
  -> declarative effect
```

For example, an `on-call-call` event is recorded against a member. A Starlark event rule may then emit an eligibility lock for that member and role; the Go planner remains responsible for applying the effect and producing the explanation trace.

These concepts remain separate:

* **Team membership**: association and authorization within a team.
* **Roster eligibility**: whether a member may be assigned to a roster.
* **Event effects**: temporary restrictions or state changes derived from recorded events.

Team membership does not imply roster eligibility. Effects target the canonical member identity, never an email address or an unresolved external identity.

## Time semantics and timezone data

Time calculations use explicit semantics:

* **Calendar days** define planning boundaries in an IANA timezone. A planning horizon of seven days means seven local calendar dates, even when a daylight-saving transition creates a 23-hour or 25-hour day.
* **Elapsed durations** define locks, cooldowns, and lifecycle durations. `24h` means exactly 24 elapsed hours.
* **Instants** define event ordering. Recorded event timestamps are stored as instants, preferably in UTC, while retaining the timezone needed to interpret local schedule boundaries.

The policy engine and its Starlark API must expose these concepts with distinct names. Calendar-day arithmetic must use timezone-aware date operations; policies must not assume that one calendar day equals 24 hours.

The selected team's IANA timezone is inherited by the policy context:

```yaml
teams:
  - id: team-a
    timezone: Europe/Berlin
```

Rules do not repeat the timezone for every calculation. The context provides it from the selected team.

The Starlark time API should expose the domain question directly:

```python
day_hours = time.calendar_day_hours(ctx.local_date)

elapsed_hours = time.elapsed_hours(start_instant, end_instant)
```

`calendar_day_hours` measures the length of one local calendar day in `ctx.team.timezone` and may return 23, 24, or 25 around daylight-saving transitions. `elapsed_hours` measures the duration between two instants, so `24h` always means exactly 24 elapsed hours. Policies should not need to inspect raw timezone transition tables.

Timezone transitions are provided by the IANA Time Zone Database. Historical transitions are required for past events, and the current database rules are sufficient for future schedules known today. Future legislation may change those rules, so published schedules must not change silently after a timezone database update.

Generated schedules should record the timezone and tzdata version used to calculate them. A changed timezone database requires an explicit schedule regeneration.

## Bounded impact decay

the impact of a duty-served on later decisions is visually represented by math plotting. Example:

```text
https://www.desmos.com/calculator/ozfmmd1ztm
```

The CLI emits structured JSON for downstream consumers; rendering and transformation remain outside the binary.

## Team-Owned Policies

For v0, Team-owned policies are git-managed.

## Configuration schema status

The configuration schema is work in progress. It currently covers the initial CLI configuration slice only; the factor lifecycle model and stricter ingress validation will be specified and implemented incrementally.

## JSON factor inspection

Inspect the current factor defaults as canonical JSON:

```text
rosterbalance inspect factors
```

The output includes a versioned schema identifier, lifecycle durations in hours, and resolved curve samples. It is intended for downstream transformation, such as a separate `gomplate` step, rather than for terminal rendering.

Render the lifecycle metadata as Graphviz DOT with the shipped template:

```sh
rosterbalance inspect factors \
  | gomplate -d config='stdin:///dev/stdin?type=application/json' \
      -f templates/factor-curve.dot.tmpl \
  | neato -n2 -Tsvg > factor-curve.svg
```

The `--decay-profile` inspection option can select any built-in profile when comparing curve outputs.

Generate diagrams for all built-in decay profiles with Mise:

```sh
mise run factor-curves
```

The generated SVGs are written to `docs/factor-curves/`.

## Math and Config

The `duty-work-served` weight uses a bounded decay profile. Each factor owns its lifecycle: the lifecycle holds the initial impact for an optional period, transitions according to its decay profile, and makes the factor irrelevant after a defined boundary.

## The core issue

| Curve       | Formula             | Hits zero?        |
| ----------- | ------------------- | ----------------- |
| Exponential | a·e^(−λt)           | No (asymptotic)   |
| Linear      | a·(1 − t/T)         | Yes, exactly at T |
| Power-law   | a·(1 − t/T)ⁿ        | Yes, exactly at T |
| Cosine      | a·(1 + cos(πt/T))/2 | Yes, exactly at T |

The first version keeps this deliberately small. The calculation is an implementation detail; user config refers to the semantic distribution of impact over the factor lifecycle.

## User-facing reference

```yaml
factors:
    - name: duty-work-served
      lifecycle:
        decay_profile: front-loaded
        hold_duration: 2d
        irrelevant_after: 14d
```

**Reading it in plain English:** *"Previous duty work keeps its initial impact for two days, then fades in a front-loaded way and is irrelevant after fourteen days."*

The lifecycle boundaries are factor configuration. A system-wide maximum may still limit `irrelevant_after`, but it does not replace the factor lifecycle.

The built-in decay profiles are:

* `hard-drop`: hold the impact until `hold_duration`, then make it zero.
* `front-loaded`: concentrate the decay near the beginning of the transition.
* `linear`: decrease the impact uniformly through the transition.
* `back-loaded`: preserve most of the impact until later in the transition.
* `flat`: do not decay before `irrelevant_after`, then make the impact zero.

The lifecycle has two distinct boundaries:

![Default duty-work-served lifecycle](docs/duty-work-served-lifecycle.svg)

This diagram illustrates the current default lifecycle. The canonical machine-readable representation remains the JSON emitted by `rosterbalance inspect factors`.

```text
0 -------- hold_duration ---------------- irrelevant_after
|          |                              |
initial    decay profile                  zero impact
impact     transition                     and irrelevant
```

* `hold_duration` keeps the initial impact unchanged for at least that long.
* `irrelevant_after` means the factor's impact is zero and must no longer affect planning.
* `decay_profile` controls the transition between those boundaries.

Lifecycle validation must ensure that durations are non-negative, `irrelevant_after` is greater than `hold_duration`, impact stays between zero and one, impact does not increase after the hold period, and impact is zero at `irrelevant_after`.

## Why the power curve is the sweet spot

* **Linear** (n=1) feels too abrupt — weight drops fast early, then stalls near zero.
* **Cosine** is smooth but hard to explain to non-math people.
* **Power-law** (n=2) gives a natural "fast at first, then tail" feel that matches intuition about memory, and the exponent is a single intuitive knob:
  * n=1 → linear
  * n=2 → "quadratic fade" (most common default)
  * n=3 → steeper initial drop

The first version may use a bounded power curve internally for the smooth profiles, while keeping the user-facing config to the five semantic profile names above. Exponents and other calculation parameters are not user-configurable.
