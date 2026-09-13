# Roster balance CLI

## Calendar ingestion

For v0, the CLI stays dumb about calendar providers and accepts normalized input through a single adapter layer. The first supported import format should be ICS exports, which lets Outlook and other corporate calendars work without committing to Graph or OAuth integration. Direct Outlook sync can come later only if the operational need justifies the extra scope.
In the future, a Starlark-based ingestion layer could map raw calendar events into roster concepts like availability, absences, or team-specific exceptions while keeping provider connectors in Go.

## Bounded impact decay

the impact of a duty-served on later decisions is visually represented by math plotting. Example:

```
https://www.desmos.com/calculator/ozfmmd1ztm
```

Also see

* [ntcharts](https://github.com/NimbleMarkets/ntcharts)
* [bubbletea](https://github.com/charmbracelet/bubbletea)

## Team-Owned Policies

For v0, Team-owned policies are git-managed.

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
