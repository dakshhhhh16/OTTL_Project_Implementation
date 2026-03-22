// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// getNilSafe provides uniform nil-propagation for OTTL path traversal.
//
// Problem: in the current OTTL codebase, accessing a missing nested key
// like body["x"]["y"] returns nil in some context packages and panics in
// others, depending on which context package handles the path resolution.
// For example, ottllog handles body access differently from ottlspan's
// attribute access. There is no consistent rule across all five context
// packages (ottllog, ottlspan, ottlmetric, ottldatapoint, ottlresource).
//
// This function exists to make that behaviour uniform across all contexts:
// if any segment of the path is missing, nil, or out of bounds, return
// pcommon.NewValueEmpty() instead of panicking. ValueTypeEmpty is the
// natural "nothing here" sentinel in the pdata type system.
//
// It is also critical for the for-range loop evaluator: when a loop body
// calls delete_key() during iteration over a snapshotted key set, fetching
// a deleted key must return a safe zero value rather than crashing.

package ottl

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// PathSegment represents one step in an OTTL path traversal.
// Either MapKey (string key for map access) or SliceIndex (integer for
// slice access) should be set, not both.
type PathSegment struct {
	MapKey     *string
	SliceIndex *int
}

// GetNilSafe traverses a pcommon.Value through the given path segments,
// returning the value at the end of the path. If ANY segment encounters
// a missing map key, a nil/empty nested value, an out-of-bounds slice
// index, or an unexpected value type, it returns pcommon.NewValueEmpty()
// instead of panicking.
//
// Handled cases:
//  1. Empty path (len(segments) == 0): return root as-is
//  2. Missing map key: return empty
//  3. Map key on non-map value: return empty
//  4. Out-of-bounds slice index: return empty
//  5. Slice index on non-slice value: return empty
//  6. Empty/nil value at any depth: propagate empty
func GetNilSafe(root pcommon.Value, segments []PathSegment) pcommon.Value {
	// Case 1: empty path — return root unchanged
	if len(segments) == 0 {
		return root
	}

	current := root

	for _, seg := range segments {
		// Case 6: empty value at any depth — stop immediately
		if current.Type() == pcommon.ValueTypeEmpty {
			return pcommon.NewValueEmpty()
		}

		if seg.MapKey != nil {
			// Map key access
			if current.Type() != pcommon.ValueTypeMap {
				// Case 3: map key on non-map value
				return pcommon.NewValueEmpty()
			}
			val, exists := current.Map().Get(*seg.MapKey)
			if !exists {
				// Case 2: missing map key
				return pcommon.NewValueEmpty()
			}
			current = val

		} else if seg.SliceIndex != nil {
			// Slice index access
			if current.Type() != pcommon.ValueTypeSlice {
				// Case 5: slice index on non-slice value
				return pcommon.NewValueEmpty()
			}
			idx := *seg.SliceIndex
			if idx < 0 || idx >= current.Slice().Len() {
				// Case 4: out-of-bounds slice index
				return pcommon.NewValueEmpty()
			}
			current = current.Slice().At(idx)

		} else {
			// TODO: handle the case where a path segment has neither MapKey
			// nor SliceIndex set — this can happen if the path expression uses
			// a computed key (e.g., attributes[some_variable]) which requires
			// the evaluator to resolve the variable first. For now, return empty.
			return pcommon.NewValueEmpty()
		}
	}

	return current
}
