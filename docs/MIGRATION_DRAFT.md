# OTTL 1.0 Migration Guide — DRAFT

> **Status:** draft — created during GSoC 2026 community bonding, not final.

## 1. Semver Commitment

Starting with OTTL 1.0, the `pkg/ottl` module follows strict semantic versioning. This means that within any 1.x release series, your existing collector configurations will continue to work without modification. Specifically:

All OTTL syntax that parses today will continue to parse in every future 1.x release. No existing statement will change its semantic meaning. The function signatures of all editors and converters in `ottlfuncs/` are frozen — their argument types, argument order, and return types will not change. New optional arguments may be added at the end of an argument list, but existing callers that do not provide the new arguments will retain their current behaviour. Path expressions defined in context packages (`ottllog`, `ottlspan`, `ottlmetric`, `ottldatapoint`, `ottlresource`, `ottlscope`, `ottlspanevent`, `ottlprofile`) are stable — a path like `attributes["key"]` or `resource.attributes["service.name"]` will continue to resolve to the same pdata field.

What OTTL 1.x reserves the right to change: internal Go APIs under `pkg/ottl/internal/` are not part of the public contract and may change in any release. The unexported AST struct types (`parsedStatement`, `editor`, `value`, etc.) are implementation details and may be refactored. Performance characteristics are informational, not contractual — while we publish benchmark targets, exact ns/op figures may vary. New features like `for-range`, new converter functions, or new context packages may be added in minor releases without breaking existing configurations.

## 2. Deprecation Timeline

| Construct | Deprecated In | Removed In | Replacement |
|-----------|--------------|------------|-------------|
| — | — | — | — |

> No constructs are deprecated in OTTL 1.0. This table will be populated during Stage 4 of the GSoC stabilization project once the comprehensive syntax audit identifies candidates for deprecation. Any deprecation will follow a minimum two-minor-release cycle: deprecated in version N, emitting a warning log; removed in version N+2 at the earliest.

## 3. `ottl.StrictTypes` Feature Gate

### What It Does

When enabled, `ottl.StrictTypes` activates a parse-time type validation pass that runs over the compiled AST before the pipeline starts processing data. This pass checks that every function argument matches the type expected by the function's `CreateDefaultArguments()` signature. Mismatches produce specific, actionable error messages at config-load time rather than silently corrupting data at runtime.

### How to Enable

Via collector configuration:
```yaml
service:
  telemetry:
    feature_gates:
      ottl.StrictTypes: true
```

Via command line:
```bash
otelcol --feature-gates=ottl.StrictTypes ...
```

### What Changes

Without `StrictTypes`, a type mismatch like `set(attributes["count"], true)` where the path expects an `int64` will silently pass at config load and produce incorrect data at runtime — the boolean `true` gets stored where an integer was expected, and downstream consumers see garbage.

With `StrictTypes` enabled, the same configuration produces this error at startup:

```
Error: failed to load config: parse-time type check failed:
  statement[3] editor "set" argument 1: expected int but got bool
```

The collector does not start until the configuration is fixed.

### Migration Steps

1. Enable `ottl.StrictTypes` in a staging or development environment first.
2. Fix any type errors reported during config load. Most common fixes involve wrapping values in explicit type conversions: `Int(...)`, `String(...)`, `Double(...)`.
3. Once all errors are resolved, enable in production.
4. The feature gate will become the default in a future 1.x minor release (timeline TBD).

## 4. FAQ

**Q: Will my existing transform processor configs break when I upgrade to OTTL 1.0?**

A: No. OTTL 1.0 is fully backward compatible with all existing configurations. The `for-range` loop construct is purely additive — it introduces new syntax that did not previously exist, and does not change the behaviour of any existing syntax. The keywords `for` and `in` are reserved in the lexer, but these words were never valid as editor names (editors must match `Lowercase(Uppercase|Lowercase)*`, meaning mixed-case names like `deleteKey` or `replaceMatch`), as converter names (converters start with an uppercase letter), or as path field names in any context package. If your configs worked before, they will work after.

**Q: What changed in the OTTL type system?**

A: The type system itself has not changed — the same eight types (string, int, float, bool, bytes, map, slice, nil) exist with the same Go and pdata mappings. What is new is enforcement: the `ottl.StrictTypes` feature gate enables compile-time type checking that catches mismatches before runtime. Previously, if you passed a `bool` where an `int64` was expected, the error would surface as silent data corruption or a runtime error (depending on the editor). With `StrictTypes`, the mismatch is caught at config load. The types themselves are unchanged; only the strictness of validation is new.

**Q: How do I migrate if I was working around the lack of looping?**

A: If you currently have large blocks of repetitive `set()` calls that copy attributes from one scope to another (like the 24-statement GKE normalization pattern from issue #30800), you can optionally replace them with a single `for-range` statement. This is entirely opt-in — your existing unrolled statements will continue to work identically with the same performance characteristics. There is no deprecation of repeated `set()` calls. If you choose to migrate, the COOKBOOK_DRAFT.md in this repository shows before/after examples for common patterns.
