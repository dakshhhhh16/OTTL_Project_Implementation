# OTTL-GSoC-2026

A standalone proof-of-work implementation for GSoC 2026 that mirrors the structure of OpenTelemetry Collector Contrib's OTTL package. The goal is to validate key pieces of an OTTL stabilization proposal, including for-range grammar support, loop-scope design, nil-safe path traversal, compatibility corpus testing, and draft design documentation.

This repository is not a fork. It is a focused experimental module to show practical, test-backed progress on language and runtime design decisions before upstream integration.

## Project structure

```text
OTTL_Project_Implementation/
├── go.mod
├── go.sum
├── README.md
├── docs/
│   ├── COOKBOOK_DRAFT.md
│   ├── MIGRATION_DRAFT.md
│   └── SPEC_DRAFT.md
└── pkg/
	└── ottl/
		├── grammar.go
		├── grammar_forrange_test.go
		├── gsoc_benchmarks_test.go
		├── nil_safe.go
		├── nil_safe_test.go
		├── ottl.go
		└── testdata/
			└── compat/
				├── README.md
				├── basic_set.ottl
				├── converter_chain.ottl
				├── gke_normalization.ottl
				└── where_clause.ottl
```

## Brief description of each part

- `go.mod`, `go.sum`: Module definition and dependency lock data.
- `pkg/ottl/grammar.go`: OTTL grammar and lexer definitions, including `parsedForRange` and reserved `for`/`in` keywords.
- `pkg/ottl/ottl.go`: Core transform context with loop-scope design (`loopScope`) and a stubbed `executeForRange` flow.
- `pkg/ottl/nil_safe.go`: `GetNilSafe` helper for safe nested path traversal without panics.
- `pkg/ottl/nil_safe_test.go`: Table-driven and no-panic tests for nil-safe behavior.
- `pkg/ottl/grammar_forrange_test.go`: Standalone participle parser tests validating for-range grammar and compatibility.
- `pkg/ottl/gsoc_benchmarks_test.go`: Benchmark stubs plus one benchmark entry-point skeleton.
- `pkg/ottl/testdata/compat/*.ottl`: Real statement corpus used for backward-compatibility parsing checks.
- `docs/SPEC_DRAFT.md`: Draft language spec (EBNF, execution notes, and type mapping).
- `docs/COOKBOOK_DRAFT.md`: Draft practical patterns and usage examples.
- `docs/MIGRATION_DRAFT.md`: Draft migration guidance and compatibility policy notes.

## Current implementation status

Implemented:

- For-range grammar AST node and lexer keyword reservation.
- Loop-scope context design and variable lookup helper.
- Nil-safe path traversal helper with broad test coverage.
- Grammar tests for for-range parser behavior.
- Compatibility corpus with real OTTL statements.
- Initial benchmark scaffold.
- Draft specification/cookbook/migration documentation.

Not fully implemented yet:

- Runtime execution for for-range evaluator.
- Full parse-time type checker pass.
- Complete benchmark implementations.
- Complete cookbook and spec examples.

## Build and test

```bash
cd OTTL_Project_Implementation
go mod tidy
go build ./...
go test ./...
```

Some tests fail by design: the getNilSafe computed-key case is a known TODO. go build ./... exits cleanly.

## Upstream references

- https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/30800 - GKE log normalisation motivating case, the for-range loop proposal.
- https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/29289 - Looping design debate, three approaches considered.
- https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/45365 - OTTL path to 1.0 stabilization tracker.
