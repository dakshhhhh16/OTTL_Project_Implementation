// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestGetNilSafe(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name     string
		root     func() pcommon.Value
		segments []PathSegment
		wantType pcommon.ValueType
		wantStr  string // expected string value when wantType is Str
	}{
		{
			name: "empty path returns root as-is",
			root: func() pcommon.Value {
				return pcommon.NewValueStr("hello")
			},
			segments: nil,
			wantType: pcommon.ValueTypeStr,
			wantStr:  "hello",
		},
		{
			name: "missing top-level map key returns empty",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				v.SetEmptyMap().PutStr("exists", "yes")
				return v
			},
			segments: []PathSegment{{MapKey: strPtr("nope")}},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "existing top-level map key returns value",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				v.SetEmptyMap().PutStr("status", "ok")
				return v
			},
			segments: []PathSegment{{MapKey: strPtr("status")}},
			wantType: pcommon.ValueTypeStr,
			wantStr:  "ok",
		},
		{
			name: "missing nested key at depth 2 returns empty",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				m := v.SetEmptyMap()
				inner := m.PutEmptyMap("level1")
				inner.PutStr("a", "found")
				return v
			},
			segments: []PathSegment{
				{MapKey: strPtr("level1")},
				{MapKey: strPtr("nonexistent")},
			},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "nil map at depth 2 - map key on string returns empty",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				v.SetEmptyMap().PutStr("level1", "not_a_map")
				return v
			},
			segments: []PathSegment{
				{MapKey: strPtr("level1")},
				{MapKey: strPtr("nested")},
			},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "out-of-bounds slice index returns empty",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				s := v.SetEmptySlice()
				s.AppendEmpty().SetStr("only_element")
				return v
			},
			segments: []PathSegment{{SliceIndex: intPtr(5)}},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "negative slice index returns empty",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				s := v.SetEmptySlice()
				s.AppendEmpty().SetStr("only")
				return v
			},
			segments: []PathSegment{{SliceIndex: intPtr(-1)}},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "valid slice index returns element",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				s := v.SetEmptySlice()
				s.AppendEmpty().SetStr("first")
				s.AppendEmpty().SetStr("second")
				return v
			},
			segments: []PathSegment{{SliceIndex: intPtr(1)}},
			wantType: pcommon.ValueTypeStr,
			wantStr:  "second",
		},
		{
			name: "slice index on non-slice returns empty",
			root: func() pcommon.Value {
				return pcommon.NewValueStr("not_a_slice")
			},
			segments: []PathSegment{{SliceIndex: intPtr(0)}},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "empty root with any segment returns empty",
			root: func() pcommon.Value {
				return pcommon.NewValueEmpty()
			},
			segments: []PathSegment{{MapKey: strPtr("anything")}},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "empty value at depth 2 stops traversal",
			root: func() pcommon.Value {
				// key "x" exists but its value is Empty — traversing deeper should return empty
				v := pcommon.NewValueEmpty()
				m := v.SetEmptyMap()
				m.PutEmpty("x")
				return v
			},
			segments: []PathSegment{
				{MapKey: strPtr("x")},
				{MapKey: strPtr("deeper")},
			},
			wantType: pcommon.ValueTypeEmpty,
		},
		{
			name: "deep traversal map -> map -> slice -> map",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				root := v.SetEmptyMap()
				l1 := root.PutEmptyMap("events")
				arr := l1.PutEmptySlice("items")
				elem := arr.AppendEmpty().SetEmptyMap()
				elem.PutStr("status", "active")
				return v
			},
			segments: []PathSegment{
				{MapKey: strPtr("events")},
				{MapKey: strPtr("items")},
				{SliceIndex: intPtr(0)},
				{MapKey: strPtr("status")},
			},
			wantType: pcommon.ValueTypeStr,
			wantStr:  "active",
		},
		{
			// will pass once computed-key branch is implemented
			name: "malformed segment returns empty",
			root: func() pcommon.Value {
				v := pcommon.NewValueEmpty()
				v.SetEmptyMap().PutStr("k", "v")
				return v
			},
			segments: []PathSegment{{}},
			wantType: pcommon.ValueTypeEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNilSafe(tt.root(), tt.segments)
			if result.Type() != tt.wantType {
				t.Errorf("type = %v, want %v", result.Type(), tt.wantType)
			}
			if tt.wantStr != "" && result.Type() == pcommon.ValueTypeStr {
				if result.Str() != tt.wantStr {
					t.Errorf("str = %q, want %q", result.Str(), tt.wantStr)
				}
			}
		})
	}
}

// TestGetNilSafeNeverPanics is the primary safety contract test.
// No combination of inputs should cause a panic.
func TestGetNilSafeNeverPanics(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	roots := []pcommon.Value{
		pcommon.NewValueEmpty(),
		pcommon.NewValueStr("s"),
		pcommon.NewValueInt(42),
		pcommon.NewValueBool(true),
		pcommon.NewValueDouble(3.14),
	}
	mapVal := pcommon.NewValueEmpty()
	mapVal.SetEmptyMap().PutStr("key", "v")
	roots = append(roots, mapVal)

	sliceVal := pcommon.NewValueEmpty()
	sliceVal.SetEmptySlice().AppendEmpty().SetStr("item")
	roots = append(roots, sliceVal)

	segments := [][]PathSegment{
		{},
		{{MapKey: strPtr("key")}},
		{{MapKey: strPtr("missing")}},
		{{SliceIndex: intPtr(0)}},
		{{SliceIndex: intPtr(-1)}},
		{{SliceIndex: intPtr(999)}},
		{{}},
		{{MapKey: strPtr("key")}, {MapKey: strPtr("nested")}},
		{{SliceIndex: intPtr(0)}, {MapKey: strPtr("field")}},
	}

	for ri, root := range roots {
		for si, segs := range segments {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC root[%d] segs[%d]: %v", ri, si, r)
					}
				}()
				_ = GetNilSafe(root, segs)
			}()
		}
	}
}
