# Rooster balance CLI

<!--
Starlark wurde ursprünglich für Konfigurations- und Build-Systeme entwickelt und ist bewusst eine deterministische, eingeschränkte Python-ähnliche Sprache. Für RosterBalance passt das konzeptionell sehr gut.

Der entscheidende Vorteil gegenüber Lua wäre für mich:

Starlark fühlt sich eher wie eine Policy-/Rule-Language an, Lua eher wie eine eingebettete Programmiersprache.

Vergleich für RosterBalance
	Eigene DSL	Starlark	Lua
Go-Einbettung	selbst bauen	gut	sehr gut
Syntax	exakt kontrollierbar	Python-artig	Lua
Ausdrucksstärke	begrenzt	hoch	sehr hoch
Determinismus	sehr gut	sehr gut	muss stärker kontrolliert werden
Sandbox	selbst bauen	Teil des Konzepts	selbst absichern
Benutzerfreundlichkeit	abhängig vom Design	gut	gut
Eigene APIs	selbst definieren	sehr gut	sehr gut
Debugging/Traceability	sehr gut machbar	gut	gut
Gefahr, zur Programmiersprache zu werden	gering	mittel	hoch

Für deine Anwendung ist insbesondere die kontrollierte Umgebung interessant.

Du könntest beispielsweise eine Starlark-Policy schreiben:

def minimum_rest(ctx):
    if ctx.role == "RB":
        return ctx.hours_since_last_assignment >= 24
    return True

def holiday_load(ctx):
    return ctx.member.holiday_assignments

Go stellt nur die Dinge zur Verfügung, die die Policy kennen darf:

ctx
├── candidate
├── member
├── slot
├── assignments
├── availability
├── workload
└── metrics

Die Policy kann also nicht plötzlich anfangen, Dateien zu lesen, HTTP Requests abzusetzen oder Deine Datenbank zu verändern.

Noch besser: Starlark muss nicht deine eigentliche Rule API sein

Ich würde sogar einen Schritt weitergehen.

Deine fachliche API sollte unabhängig von Starlark sein:

type Rule interface {
    ID() string
    Evaluate(Context) Evaluation
}

Dann hast du einen Starlark-Adapter:

             ┌───────────────┐
             │ Starlark file │
             └───────┬───────┘
                     │
                     ▼
              Starlark Runtime
                     │
                     ▼
              Policy Adapter
                     │
                     ▼
                  Rule API
                     │
                     ▼
             Planning Engine

Damit bleibt dir sogar die Möglichkeit, später zu sagen:

default rules     → deklarativ
complex rules     → Starlark
built-in rules    → Go

ohne dass der Planner davon etwas wissen muss.

Ein wichtiger Punkt für deine Suche

Ich würde nicht die komplette Bewertungslogik in Starlark schreiben.

Also nicht:

def choose_next_member(ctx):
    ...
    # kompletter Search Algorithmus

Sondern:

def constraint(ctx):
    ...

def score(ctx):
    ...

Der Search Engine gehört in Go.

Das ergibt:

Starlark
  │
  │ "Ist dieser Kandidat erlaubt?"
  │ "Wie teuer ist dieser Kandidat?"
  ▼
Rule Runtime
  │
  ▼
Go Search Engine
  │
  ├── state
  ├── candidate generation
  ├── pruning
  ├── backtracking / beam search / ...
  └── best solution

Das halte ich für eine sehr saubere Trennung.

Und ich würde Starlark tatsächlich ausprobieren

Nicht sofort die gesamte DSL bauen.

Mach einen kleinen Spike:

rosterbalance
├── Go domain
├── Go planner
├── Starlark runtime
└── 3 Regeln
    ├── availability
    ├── minimum_rest
    └── holiday_load

Dann prüfst du drei Dinge:

Kann ich die Regeln angenehm formulieren?
Kann ich aus jeder Rule Evaluation eine gute Erklärung erzeugen?
Ist die Performance bei tausenden/millionen Evaluierungen akzeptabel?

Gerade Punkt 3 ist wichtig: Deine Search Engine könnte dieselbe Regel extrem oft auswerten. Deshalb würde ich Starlark zunächst nur als Policy-Layer betrachten und nicht voraussetzen, dass jede einzelne Low-Level-Berechnung über den Interpreter laufen muss.

Mein aktueller Favorit wäre deshalb:

Go für Domain + Planning + Search, Starlark für Policies/Rules, CLI als erste Application-Schicht.

Das ist für RosterBalance meiner Meinung nach eine ziemlich starke Architektur.

gut! die drei regeln sollen erweiterbar sein, und es soll eine art pluginsystem geben. bitte gib mir dazu ein konzep

Ja. Ich würde das Pluginsystem nicht als klassische Go-Plugin-Architektur bauen. Für RosterBalance ist ein fachliches Plugin-System auf Basis von Starlark deutlich passender.

Der zentrale Gedanke:

Go definiert die stabile Runtime und die Fähigkeiten des Systems. Plugins definieren Regeln und Policies.

Damit können Nutzer RosterBalance erweitern, ohne den Go-Core neu zu kompilieren.

1. Zielarchitektur
                         RosterBalance CLI
                               │
                               ▼
                       ┌───────────────┐
                       │  Application  │
                       │    Layer      │
                       └───────┬───────┘
                               │
                               ▼
                       ┌───────────────┐
                       │    Planner    │
                       └───────┬───────┘
                               │
             ┌─────────────────┼─────────────────┐
             │                 │                 │
             ▼                 ▼                 ▼
        Constraints         Metrics          Scoring
             │                 │                 │
             └─────────────────┼─────────────────┘
                               │
                               ▼
                       ┌───────────────┐
                       │ Policy/Rule   │
                       │    Runtime    │
                       └───────┬───────┘
                               │
                       ┌───────┴────────┐
                       ▼                ▼
                 Built-in Rules    Plugins
                       │                │
                       │          ┌─────┴─────┐
                       │          │ Starlark  │
                       │          │ Policies  │
                       │          └───────────┘
                       ▼
                  Decision Trace

Dabei sind die ursprünglichen drei Regeln keine Sonderfälle mehr. Sie sind einfach die ersten drei Implementierungen eines allgemeinen Rule-Systems.

2. Drei primitive Plugin-Typen

Ich würde zunächst genau drei fachliche Plugin-Typen definieren:

Constraint

Beantwortet:

Darf dieser Kandidat überhaupt gewählt werden?

availability
minimum_rest
qualification
...

Ergebnis:

type ConstraintResult struct {
    Passed      bool
    Explanation string
}
Metric

Beantwortet:

Welche Eigenschaft hat dieser Kandidat bzw. dieser Zustand?

Beispiele:

historical_load
holiday_load
days_since_last_rb
assignments_in_current_period
type MetricResult struct {
    Value float64
}
Scoring Rule

Beantwortet:

Wie stark soll diese Eigenschaft die Entscheidung beeinflussen?

Zum Beispiel:

holiday_load × 50
historical_load_deviation × 100

Das ist wichtig, weil dadurch Metric und Gewicht getrennt bleiben.

3. Plugins sollten deklarieren, was sie anbieten

Ich würde jedem Plugin eine kleine Manifest-Datei geben.

Zum Beispiel:

plugins/
└── fairness/
    ├── plugin.toml
    └── policy.star

plugin.toml:

id = "rosterbalance.fairness"
version = "1.0.0"
api_version = "1"

[plugin]
name = "Fairness Rules"
description = "Fairness and workload balancing rules"

[[rules]]
id = "holiday-load"
kind = "metric"

[[rules]]
id = "historical-load"
kind = "metric"

[[rules]]
id = "balance-holiday-load"
kind = "scoring"

Damit kann Go das Plugin inspizieren, validieren und registrieren, bevor überhaupt Starlark ausgeführt wird.

4. Starlark implementiert die Regeln

Beispielsweise:

def holiday_load(ctx):
    return ctx.member.holiday_assignments

def historical_load(ctx):
    return ctx.member.historical_assignments

def balance_holiday_load(ctx):
    return holiday_load(ctx) * 50.0

Aber ich würde nicht einfach beliebige Funktionen aus dem Modul suchen.

Besser wäre eine explizite Registration:

metric(
    id = "holiday-load",
    evaluate = holiday_load,
)

metric(
    id = "historical-load",
    evaluate = historical_load,
)

score(
    id = "balance-holiday-load",
    evaluate = balance_holiday_load,
)

Das gibt dir eine klare Plugin-Schnittstelle.

5. Der Plugin Context ist die wichtigste Grenze

Starlark sollte niemals direkt auf dein Domain-Objekt zugreifen können.

Also nicht:

ctx.member.__internal_state

sondern eine bewusst definierte API:

ctx
├── candidate
│   ├── member
│   ├── role
│   └── slot
│
├── member
│   ├── current_load
│   ├── historical_load
│   └── holiday_load
│
├── history
│   ├── last_assignment
│   ├── assignments
│   └── assignments_for_role()
│
├── availability
│   └── is_available()
│
└── metrics
    └── get()

Das ist im Grunde dein Plugin SDK.

Und diese API kannst du später versionieren.

6. Plugins dürfen nicht den Planner kontrollieren

Das würde ich als harte Architekturregel festlegen.

Ein Plugin darf:

lesen:
    Context

berechnen:
    Constraint
    Metric
    Score

erklären:
    Explanation

Ein Plugin darf nicht:

✗ State verändern
✗ Assignments erzeugen
✗ andere Plugins manipulieren
✗ Datenbankzugriff
✗ Netzwerkzugriff
✗ Filesystemzugriff
✗ Search Strategy verändern

Der Planner bleibt vollständig unter Kontrolle des Go-Cores.

Das macht das System wesentlich vorhersehbarer.

7. Search ist ebenfalls erweiterbar – aber anders

Hier würde ich einen wichtigen Unterschied machen.

Du könntest später Plugins für:

Search Strategy
├── greedy
├── backtracking
├── beam-search
├── branch-and-bound
└── ...

haben.

Aber diese würde ich zunächst als Go-Komponenten behandeln.

Also:

type SearchStrategy interface {
    Search(problem PlanningProblem) PlanningResult
}

Warum?

Weil Search sehr viel häufiger und tiefer mit dem State interagiert als eine normale Rule.

Starlark eignet sich hervorragend für:

„Bewerte diesen Zustand/Kandidaten.“

Weniger gut als:

„Implementiere unseren kompletten hochperformanten Suchalgorithmus.“

8. Plugin Lifecycle

Ich würde den Lifecycle ungefähr so definieren:

Discovery
   ↓
Load manifest
   ↓
Validate manifest
   ↓
Check API version
   ↓
Load Starlark
   ↓
Register rules
   ↓
Validate registrations
   ↓
Plugin Registry
   ↓
Planning

Die Registry könnte intern so aussehen:

type Registry struct {
    Constraints map[string]Constraint
    Metrics     map[string]Metric
    Scoring     map[string]ScoringRule
}

Dann wird der Planner komplett unabhängig davon:

constraints := registry.Constraints
metrics := registry.Metrics
scoring := registry.Scoring
9. Policy entscheidet, welche Plugins benutzt werden

Das ist ein weiterer wichtiger Punkt.

Installiert bedeutet nicht automatisch aktiv.

Zum Beispiel:

[policy]
id = "team-default"

[[constraints]]
rule = "core.availability"

[[constraints]]
rule = "core.minimum-rest"

[[constraints]]
rule = "company.qualification"

[[metrics]]
rule = "rosterbalance.fairness.holiday-load"
weight = 50

[[metrics]]
rule = "rosterbalance.fairness.historical-load"
weight = 100

Damit kannst du verschiedene Policies für verschiedene Teams haben.

Plugin
   ↓
liefert Regeln

Policy
   ↓
kombiniert Regeln

Planner
   ↓
führt Policy aus

Das ist eine sehr schöne Trennung.

10. Explainability wird Teil des Plugin Contracts

Das würde ich bei RosterBalance sogar zur Pflicht für Plugins machen.

Ein Plugin liefert nicht nur:

42.0

sondern:

rule: holiday-load
value: 3
weight: 50
contribution: 150

explanation:
"Anna has 3 holiday assignments in the relevant history."

Bei Constraints:

rule: minimum-rest
passed: false

explanation:
"Bob had an assignment ending at 18:00.
The candidate starts at 06:00.
Required rest: 24h.
Actual rest: 12h."

Damit bleibt die wichtigste Eigenschaft von RosterBalance erhalten, auch wenn jemand das System durch Plugins erweitert.

11. Versionierung würde ich von Anfang an vorsehen

Nicht nur:

version = "1.0"

sondern zwei Versionen:

version = "1.3.0"
api_version = "1"

api_version bedeutet:

Welche Plugin-Schnittstelle erwartet das Plugin?

version bedeutet:

Welche Version dieses Plugins ist das?

Damit kannst du später beispielsweise:

RosterBalance Plugin API 1
RosterBalance Plugin API 2

parallel unterstützen oder sauber inkompatible Plugins ablehnen.

12. Was ich ausdrücklich vermeiden würde

Ich würde kein System bauen, bei dem man einfach irgendeine .so/.dll in ein Plugin-Verzeichnis legt.

Also nicht:

plugins/
  myplugin.so

mit einer Go-Plugin-Schnittstelle.

Das hat für RosterBalance einige Nachteile:

OS-/Architekturabhängigkeit
Deployment komplizierter
Go-Version-Kompatibilität
schwerer zu versionieren
Sicherheitsmodell schwieriger
viel stärkere Kopplung an den Core

Starlark gibt dir dagegen eine ziemlich elegante Grenze:

             Go
              │
       stable Plugin API
              │
        ──────┼──────
              │
           Starlark
Mein Vorschlag für RosterBalance v0.1

Ich würde das bewusst klein halten:

Core
├── Domain
├── Planning Problem
├── State
├── Candidate Generator
├── Constraint Engine
├── Metric Engine
├── Scoring Engine
├── Search Engine
└── Decision Trace

Plugin System
├── Plugin Manifest
├── Plugin Loader
├── Plugin Registry
├── Starlark Runtime
└── Plugin Context

Built-in Plugins
├── core.availability
├── core.minimum-rest
└── core.qualification

External example plugin
└── fairness
    ├── holiday-load
    └── historical-load

Damit hast du bereits ein echtes Erweiterungssystem, ohne dass du dich in einem riesigen Framework verlierst.

Und ich würde sogar einen Architekturgrundsatz in die Spezifikation schreiben:

Everything that influences planning decisions must be representable as a registered, identifiable rule and must produce an explainable evaluation result.

Das wäre für mich einer der zentralen Design-Sätze von RosterBalance.
-->
