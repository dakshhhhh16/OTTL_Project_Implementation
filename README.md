# ottl-gsoc-2026

A standalone proof-of-work implementation I built during GSoC 2026 community bonding to demonstrate that I actually understand the internals of `pkg/ottl/` in [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib). This is not a fork — it's a self-contained Go module that mirrors the real package structure and implements the foundational pieces of the OTTL stabilization proposal.

The grammar types are copied verbatim from the real `grammar.go` with their exact participle/v2 struct tags. The lexer is the real lexer with two additional reserved keywords. The `TransformContext` shows where `loopScope` will live and why. The `GetNilSafe` helper is a real partial implementation that handles the five most common nil-propagation cases. The tests parse real OTTL syntax and the testdata corpus contains real statements from the transform processor README and issue #30800.

## What is implemented

- **`pkg/ottl/grammar.go`** — All grammar structs from the real codebase (parsedStatement, editor, converter, value, path, booleanExpression, mathExpression, etc.) with exact participle tags. New `parsedForRange` struct placed after `parsedStatement`. Reserved `for`/`in` keywords in the lexer with comments explaining why the reservation is safe.
- **`pkg/ottl/ottl.go`** — `TransformContext` with `loopScope map[string]pcommon.Value` field, `GetLoopVar()` for path resolution, and `executeForRange()` stub with a detailed 6-step implementation plan in comments.
- **`pkg/ottl/nil_safe.go`** — `GetNilSafe()` handling 5 of 6 cases with real working code. The computed-key case is left as an explicit TODO.
- **`pkg/ottl/nil_safe_test.go`** — 12 table-driven test cases plus a no-panic safety test.
- **`pkg/ottl/grammar_forrange_test.go`** — Standalone participle parser test for `parsedForRange` (4 cases) plus backward-compatibility test for existing statements (6 statements).
- **`pkg/ottl/gsoc_benchmarks_test.go`** — 5 benchmark stubs in `package ottl_test`.
- **`pkg/ottl/testdata/compat/`** — 4 `.ottl` corpus files including the 24-statement GKE normalization from #30800.
- **`docs/`** — SPEC_DRAFT.md (EBNF + type table + execution model), COOKBOOK_DRAFT.md (GKE + PII examples), MIGRATION_DRAFT.md (semver + StrictTypes + FAQ).

## What is stubbed

- For-range evaluation (`executeForRange` is a stub with TODO)
- `ParseTimeTypeChecker` AST pass (Stage 2 deliverable)
- Benchmark implementations for `ForRangeMap100`, `ForRangeSlice1000`, `NestedForRange`
- Remaining 6 cookbook examples (headings only)
- Error taxonomy examples in the spec

## Build

```bash
cd OTTL_Project_implementation
go mod tidy
go build ./...
go test ./...
```

## Related upstream issues

- [#30800](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/30800) — for-range loop proposal (the GKE normalization problem)
- [#29289](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/29289) — OTTL stabilization tracking
- [#45365](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/45365) — type system inconsistencies
