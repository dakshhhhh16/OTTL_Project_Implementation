// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl

import (
	"testing"
)

// TestParsedForRangeStruct builds a standalone participle parser for
// parsedForRange and validates it against representative OTTL for-range
// inputs. Since parsedForRange is not yet wired into the main
// parsedStatement union, building a standalone parser is the correct
// approach — it proves the grammar design and participle tags are valid.
func TestParsedForRangeStruct(t *testing.T) {
	parser := newForRangeParser()

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantKeyVar string
		wantValVar string
		wantWhere  bool // true if Where clause should be non-nil
		wantBody   int  // expected number of body statements (0 if wantErr)
	}{
		{
			name:       "basic map loop",
			input:      `for key, val in attributes { set(resource.attributes["env"], val) }`,
			wantKeyVar: "key",
			wantValVar: "val",
			wantWhere:  false,
			wantBody:   1,
		},
		{
			name:       "basic slice loop with indexed path",
			input:      `for i, item in body["events"] { set(item, "processed") }`,
			wantKeyVar: "i",
			wantValVar: "item",
			wantWhere:  false,
			wantBody:   1,
		},
		{
			name:       "where-guarded loop",
			input:      `for k, v in attributes where k != "password" { set(resource.attributes[k], v) }`,
			wantKeyVar: "k",
			wantValVar: "v",
			wantWhere:  true,
			wantBody:   1,
		},
		{
			name:    "missing in keyword - should fail to parse",
			input:   `for key, val attributes { set(resource.attributes["x"], val) }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseString("", tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected parse error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected parse error for %q: %v", tt.input, err)
			}

			if result.KeyVar != tt.wantKeyVar {
				t.Errorf("KeyVar = %q, want %q", result.KeyVar, tt.wantKeyVar)
			}
			if result.ValVar != tt.wantValVar {
				t.Errorf("ValVar = %q, want %q", result.ValVar, tt.wantValVar)
			}
			if tt.wantWhere && result.Where == nil {
				t.Error("Where should be non-nil for where-guarded input")
			}
			if !tt.wantWhere && result.Where != nil {
				t.Error("Where should be nil when no where clause is present")
			}
			if len(result.Body) != tt.wantBody {
				t.Errorf("Body has %d statements, want %d", len(result.Body), tt.wantBody)
			}
		})
	}
}

// TestExistingStatementsStillParse verifies that the reserved for/in keywords
// do not break parsing of existing OTTL statements. This is the backward-
// compatibility guarantee.
func TestExistingStatementsStillParse(t *testing.T) {
	parser := newStatementParser()

	// These are real OTTL statements from the transform processor README.
	// They MUST all continue to parse correctly after we reserve for/in.
	statements := []string{
		`set(attributes["key"], "value")`,
		`set(attributes["count"], 42)`,
		`set(resource.attributes["env"], "prod") where attributes["level"] == "error"`,
		`delete_key(attributes, "password")`,
		`keep_keys(attributes, ["http.method", "http.status_code"])`,
		`merge_maps(attributes, attributes, "upsert")`,
	}

	for _, stmt := range statements {
		t.Run(stmt, func(t *testing.T) {
			_, err := parser.ParseString("", stmt)
			if err != nil {
				t.Errorf("failed to parse existing statement %q: %v", stmt, err)
			}
		})
	}
}
