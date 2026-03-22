// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ---------------------------------------------------------------------------
// TransformContext — mirrors the shared context concept from pkg/ottl/
// ---------------------------------------------------------------------------

// TransformContext holds the telemetry data being transformed and any
// per-execution state. In the real codebase, each context package
// (ottllog, ottlspan, ottlmetric, ottldatapoint, ottlresource, ottlscope,
// ottlspanevent, ottlprofile) defines its own TransformContext that
// implements the generic constraint required by Statement[K any].
//
// This proof-of-work mirrors the shared design concept and adds the
// loopScope field that the proposal introduces.
type TransformContext struct {
	resource pcommon.Resource
	scope    pcommon.InstrumentationScope

	// Note: the real TransformContext also carries a cache pcommon.Map for
	// inter-statement intermediate values. Omitted here for brevity.

	// loopScope holds the variables bound by a for-range loop for the current iteration.
	// It is nil for all non-loop statement executions — the nil check is free and means
	// existing code paths pay exactly zero cost for the loop feature existing.
	// Allocated once per for-range execution, reset between iterations via clear(loopScope)
	// rather than reallocated — this is the Go 1.21 pattern for zero per-iteration heap
	// allocations on a hot path. Path resolution checks loopScope first before falling
	// back to the normal pdata accessor chain, which means loop variables are visible
	// inside the body without modifying any of the five context packages.
	loopScope map[string]pcommon.Value
}

// NewTransformContext creates a new TransformContext.
// loopScope is intentionally left nil — it is allocated on first for-range entry.
func NewTransformContext(resource pcommon.Resource, scope pcommon.InstrumentationScope) TransformContext {
	return TransformContext{
		resource: resource,
		scope:    scope,
	}
}

// GetResource returns the resource associated with the telemetry data.
func (ctx TransformContext) GetResource() pcommon.Resource {
	return ctx.resource
}

// GetInstrumentationScope returns the instrumentation scope.
func (ctx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return ctx.scope
}

// GetLoopVar looks up a variable in the current loop scope.
// Returns the value and true if found, or an empty value and false if:
//   - We are not inside a for-range loop (loopScope is nil)
//   - The variable name is not bound in the current iteration
//
// This is called during path resolution BEFORE falling back to the normal
// pdata accessor chain. Because the nil check on a map is free (just a
// pointer compare), non-loop code paths pay zero cost.
//
// Note: pcommon.Value is a reference type that wraps a shared underlying
// pdata storage, so returning a copy is safe for mutation purposes — the
// caller modifies the same underlying data.
func (ctx *TransformContext) GetLoopVar(name string) (pcommon.Value, bool) {
	if ctx.loopScope == nil {
		return pcommon.NewValueEmpty(), false
	}
	val, ok := ctx.loopScope[name]
	return val, ok
}

// ---------------------------------------------------------------------------
// executeForRange — stub for the for-range evaluator
// ---------------------------------------------------------------------------

// executeForRange is the evaluation entry point for a parsedForRange node.
// It is not yet implemented — this stub outlines the exact steps that the
// implementation will follow, matching the architecture of the existing
// Statement.Execute in pkg/ottl/expression.go.
//
// Implementation plan (ordered steps):
//  1. Resolve Target via the getter compiled from the parsedForRange.Target
//     value node. This produces a pcommon.Value at runtime.
//  2. Type-assert the resolved value to pdata.Map or pdata.Slice. If neither,
//     return an error (for-range only supports iterable types).
//  3. For Map iteration: snapshot the keys into a []string BEFORE entering the
//     loop. This prevents concurrent-modification panics if the body deletes
//     or adds keys during iteration.
//  4. Allocate loopScope once: ctx.loopScope = make(map[string]pcommon.Value, 2)
//  5. Per iteration:
//     a. clear(ctx.loopScope)  — resets without deallocating
//     b. ctx.loopScope[keyVar] = current key (string for Map, int for Slice)
//     c. ctx.loopScope[valVar] = current value
//     d. If Where clause exists, evaluate it. If false, skip to next iteration.
//     e. Execute all Body statements in declaration order using Statement.Execute
//  6. After loop exits: ctx.loopScope = nil (restores zero-cost for subsequent
//     non-loop statements)
//
// Error handling follows the same error_mode pattern as StatementSequence:
//   - propagate: log and return the error immediately
//   - ignore: log a warning and continue to the next iteration
//   - silent: discard the error entirely
func executeForRange(ctx *TransformContext, fr *parsedForRange) error {
	// TODO: implement once the getter compilation pipeline is wired up
	_ = ctx
	_ = fr
	return fmt.Errorf("for-range evaluation is not yet implemented — this is a GSoC 2026 deliverable")
}
