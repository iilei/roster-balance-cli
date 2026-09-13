# Rooster balance CLI

~~~
rosterbalance/
├── cmd/
│   └── rosterbalance/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── member.go
│   │   ├── roster.go
│   │   ├── assignment.go
│   │   ├── availability.go
│   │   └── role.go
│   │
│   ├── planning/
│   │   ├── problem.go
│   │   ├── state.go
│   │   ├── candidate.go
│   │   ├── planner.go
│   │   ├── constraints/
│   │   ├── metrics/
│   │   ├── scoring/
│   │   └── search/
│   │       ├── strategy.go
│   │       ├── greedy.go
│   │       └── backtracking.go
│   │
│   ├── config/
│   │   └── policy.go
│   │
│   └── io/
│       ├── input.go
│       └── output.go
│
├── policy/
│   └── default.toml
│
├── examples/
│   └── small-team.toml
│
├── docs/
│   ├── specification.md
│   └── adr/
│
├── testdata/
└── go.mod
~~~
