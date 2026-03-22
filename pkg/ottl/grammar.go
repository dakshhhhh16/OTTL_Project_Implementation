// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package ottl mirrors the structure of pkg/ottl in opentelemetry-collector-contrib.
// This is a standalone proof-of-work implementation for GSoC 2026, not a fork.
package ottl

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// ---------------------------------------------------------------------------
// Grammar types — copied verbatim from pkg/ottl/grammar.go
// ---------------------------------------------------------------------------

// parsedStatement represents a parsed statement. It is the entry point into the statement DSL.
type parsedStatement struct {
	Editor editor `parser:"(@@"`
	// If converter is matched then return error
	Converter   *converter         `parser:"|@@)"`
	WhereClause *booleanExpression `parser:"( 'where' @@ )?"`
}

// constExpr is a boolean constant or converter result used in boolean expressions.
type constExpr struct {
	Boolean   *boolean   `parser:"( @Boolean"`
	Converter *converter `parser:"| @@ )"`
}

// booleanValue represents something that evaluates to a boolean --
// either an equality or inequality, explicit true or false, or
// a parenthesized subexpression.
type booleanValue struct {
	Negation   *string            `parser:"@OpNot?"`
	Comparison *comparison        `parser:"( @@"`
	ConstExpr  *constExpr         `parser:"| @@"`
	SubExpr    *booleanExpression `parser:"| '(' @@ ')' )"`
}

// opAndBooleanValue represents the right side of an AND boolean expression.
type opAndBooleanValue struct {
	Operator string        `parser:"@OpAnd"`
	Value    *booleanValue `parser:"@@"`
}

// term represents an arbitrary number of boolean values joined by AND.
type term struct {
	Left  *booleanValue        `parser:"@@"`
	Right []*opAndBooleanValue `parser:"@@*"`
}

// opOrTerm represents the right side of an OR boolean expression.
type opOrTerm struct {
	Operator string `parser:"@OpOr"`
	Term     *term  `parser:"@@"`
}

// booleanExpression represents a true/false decision expressed
// as an arbitrary number of terms separated by OR.
type booleanExpression struct {
	Left  *term       `parser:"@@"`
	Right []*opOrTerm `parser:"@@*"`
}

// compareOp is the type of a comparison operator.
type compareOp int

const (
	eq compareOp = iota
	ne
	lt
	lte
	gte
	gt
)

var compareOpTable = map[string]compareOp{
	"==": eq, "!=": ne, "<": lt, "<=": lte, ">": gt, ">=": gte,
}

func (c *compareOp) Capture(values []string) error {
	op, ok := compareOpTable[values[0]]
	if !ok {
		return fmt.Errorf("'%s' is not a valid operator", values[0])
	}
	*c = op
	return nil
}

// comparison represents an optional boolean condition.
type comparison struct {
	Left  value     `parser:"@@"`
	Op    compareOp `parser:"@OpComparison"`
	Right value     `parser:"@@"`
}

// editor represents the function call of a statement.
type editor struct {
	Function  string     `parser:"@(Lowercase(Uppercase | Lowercase)*)"`
	Arguments []argument `parser:"'(' ( @@ ( ',' @@ )* )? ')'"`
	// If keys are matched return an error
	Keys []key `parser:"( @@ )*"`
}

// converter represents a converter function call.
type converter struct {
	Function  string     `parser:"@(Uppercase(Uppercase | Lowercase)*)"`
	Arguments []argument `parser:"'(' ( @@ ( ',' @@ )* )? ')'"`
	Keys      []key      `parser:"( @@ )*"`
}

type argument struct {
	Name         string  `parser:"(@(Lowercase(Uppercase | Lowercase)*) Equal)?"`
	Value        value   `parser:"( @@"`
	FunctionName *string `parser:"| @(Uppercase(Uppercase | Lowercase)*) )"`
}

// value represents a part of a parsed statement which is resolved to a value of some sort.
// This can be a telemetry path, mathExpression, function call, or literal.
type value struct {
	IsNil          *isNil           `parser:"( @Nil"`
	Literal        *mathExprLiteral `parser:"| @@ (?! OpAddSub | OpMultDiv)"`
	MathExpression *mathExpression  `parser:"| @@"`
	Bytes          *byteSlice       `parser:"| @Bytes"`
	String         *string          `parser:"| @String"`
	Bool           *boolean         `parser:"| @Boolean"`
	Enum           *enumSymbol      `parser:"| @Uppercase (?! Lowercase)"`
	Map            *mapValue        `parser:"| @@"`
	List           *list            `parser:"| @@)"`
}

// path represents a telemetry path expression.
type path struct {
	Pos     lexer.Position
	Context string  `parser:"(@Lowercase '.')?"`
	Fields  []field `parser:"@@ ( '.' @@ )*"`
}

type field struct {
	Name string `parser:"@Lowercase"`
	Keys []key  `parser:"( @@ )*"`
}

type key struct {
	String         *string          `parser:"'[' (@String "`
	Int            *int64           `parser:"| @Int"`
	MathExpression *mathExpression  `parser:"| @@"`
	Expression     *mathExprLiteral `parser:"| @@ ) ']'"`
}

type list struct {
	Values []value `parser:"'[' (@@)* (',' @@)* ']'"`
}

type mapValue struct {
	Values []mapItem `parser:"'{' (@@ ','?)* '}'"`
}

type mapItem struct {
	Key   *string `parser:"@String ':'"`
	Value *value  `parser:"@@"`
}

type byteSlice []byte

func (b *byteSlice) Capture(values []string) error {
	rawStr := values[0][2:]
	newBytes, err := hex.DecodeString(rawStr)
	if err != nil {
		return err
	}
	*b = newBytes
	return nil
}

type boolean bool

func (b *boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type isNil bool

func (n *isNil) Capture(_ []string) error {
	*n = true
	return nil
}

type mathExprLiteral struct {
	// If editor is matched then error
	Editor    *editor    `parser:"( @@"`
	Converter *converter `parser:"| @@"`
	Float     *float64   `parser:"| @Float"`
	Int       *int64     `parser:"| @Int"`
	Path      *path      `parser:"| @@ )"`
}

type mathValue struct {
	UnaryOp       *mathOp          `parser:"@OpAddSub?"`
	Literal       *mathExprLiteral `parser:"( @@"`
	SubExpression *mathExpression  `parser:"| '(' @@ ')' )"`
}

type opMultDivValue struct {
	Operator mathOp     `parser:"@OpMultDiv"`
	Value    *mathValue `parser:"@@"`
}

type addSubTerm struct {
	Left  *mathValue        `parser:"@@"`
	Right []*opMultDivValue `parser:"@@*"`
}

type opAddSubTerm struct {
	Operator mathOp      `parser:"@OpAddSub"`
	Term     *addSubTerm `parser:"@@"`
}

type mathExpression struct {
	Left  *addSubTerm     `parser:"@@"`
	Right []*opAddSubTerm `parser:"@@*"`
}

type mathOp int

const (
	add mathOp = iota
	sub
	mult
	div
)

var mathOpTable = map[string]mathOp{
	"+": add, "-": sub, "*": mult, "/": div,
}

func (m *mathOp) Capture(values []string) error {
	op, ok := mathOpTable[values[0]]
	if !ok {
		return fmt.Errorf("'%s' is not a valid operator", values[0])
	}
	*m = op
	return nil
}

type enumSymbol string

// ---------------------------------------------------------------------------
// parsedForRange — NEW for GSoC 2026
// ---------------------------------------------------------------------------

// parsedForRange is the AST node for a for-range loop over a pdata.Map or pdata.Slice.
// It is not yet wired into the parsedStatement union — that comes once the RFC is approved
// and the scoping model is finalised. For now it exists as a standalone compiled type
// to prove the grammar design is valid and the participle tags are correct.
//
// Grammar:
//   for <keyVar>, <valVar> in <iterable> [where <condition>] { <body>+ }
//
// Design notes:
//   - KeyVar/ValVar use the Lowercase token, matching Go's own range idiom.
//   - The optional Where clause reuses the existing booleanExpression evaluator.
//   - Body is []*parsedStatement so the loop body is standard OTTL statements.
//   - The 'for' keyword is unambiguous because editor names start with
//     Lowercase(Uppercase|Lowercase)* and "for" is now a reserved For token.
type parsedForRange struct {
	KeyVar string             `parser:"'for' @Lowercase ','"`
	ValVar string             `parser:"@Lowercase 'in'"`
	Target value              `parser:"@@"`
	Where  *booleanExpression `parser:"( 'where' @@ )?"`
	Body   []*parsedStatement `parser:"'{' @@* '}'"`
}

// ---------------------------------------------------------------------------
// Lexer — buildLexer() copied from pkg/ottl/grammar.go with additions
// ---------------------------------------------------------------------------

// buildLexer constructs a SimpleLexer definition.
// Note that the ordering of these rules matters.
// It's in a separate function so it can be easily tested alone (see lexer_test.go).
func buildLexer() *lexer.StatefulDefinition {
	return lexer.MustSimple([]lexer.SimpleRule{
		{Name: `Bytes`, Pattern: `0x[a-fA-F0-9]+`},
		{Name: `Float`, Pattern: `(\d+\.\d*|\d*\.\d+)([eE][-+]?\d+)?`},
		{Name: `Int`, Pattern: `\d+`},
		{Name: `String`, Pattern: `"(\\.|[^\\"])*"`},
		{Name: `Nil`, Pattern: `\b(nil)\b`},
		{Name: `OpNot`, Pattern: `\b(not)\b`},
		{Name: `OpOr`, Pattern: `\b(or)\b`},
		{Name: `OpAnd`, Pattern: `\b(and)\b`},
		{Name: `OpComparison`, Pattern: `==|!=|>=|<=|>|<`},
		{Name: `OpAddSub`, Pattern: `\+|\-`},
		{Name: `OpMultDiv`, Pattern: `\/|\*`},
		// Reserved keywords for the for-range loop construct (GSoC 2026).
		// These are safe to reserve because:
		//   - Editor names are validated by grammarCustomErrorsVisitor to match
		//     the pattern Lowercase(Uppercase|Lowercase)*, meaning they must start
		//     with a lowercase letter followed by mixed-case alphanumeric. "for"
		//     and "in" are pure lowercase and were never valid as standalone editor
		//     names in any released version of OTTL.
		//   - Converter names must start with Uppercase, so no conflict there.
		//   - No path field in any context package (ottllog, ottlspan, ottlmetric,
		//     ottldatapoint, ottlresource, ottlscope, ottlspanevent, ottlprofile)
		//     uses "for" or "in" as a field name.
		// Keyword reservation is clean — no grammar disambiguation needed.
		{Name: `For`, Pattern: `\b(for)\b`},
		{Name: `In`, Pattern: `\b(in)\b`},
		{Name: `Boolean`, Pattern: `\b(true|false)\b`},
		{Name: `Equal`, Pattern: `=`},
		{Name: `LParen`, Pattern: `\(`},
		{Name: `RParen`, Pattern: `\)`},
		{Name: `LBrace`, Pattern: `\{`},
		{Name: `RBrace`, Pattern: `\}`},
		{Name: `Colon`, Pattern: `\:`},
		{Name: `Punct`, Pattern: `[,.\[\]]`},
		{Name: `Uppercase`, Pattern: `[A-Z][A-Z0-9_]*`},
		{Name: `Lowercase`, Pattern: `[a-z][a-z0-9_]*`},
		{Name: "whitespace", Pattern: `\s+`},
	})
}

// ---------------------------------------------------------------------------
// grammarCustomError — copied from pkg/ottl/grammar.go
// ---------------------------------------------------------------------------

type grammarCustomError struct {
	errs []error
}

func (e *grammarCustomError) Error() string {
	switch len(e.errs) {
	case 0:
		return ""
	case 1:
		return e.errs[0].Error()
	default:
		var b strings.Builder
		b.WriteString(e.errs[0].Error())
		for _, err := range e.errs[1:] {
			b.WriteString("; ")
			b.WriteString(err.Error())
		}
		return b.String()
	}
}

// ---------------------------------------------------------------------------
// Parser builders
// ---------------------------------------------------------------------------

// newStatementParser builds a participle parser for parsedStatement.
// This mirrors the real newParser[G any]() in pkg/ottl/parser.go.
func newStatementParser() *participle.Parser[parsedStatement] {
	p, err := participle.Build[parsedStatement](
		participle.Lexer(buildLexer()),
		participle.Unquote("String"),
		participle.Elide("whitespace"),
		participle.UseLookahead(participle.MaxLookahead),
	)
	if err != nil {
		panic("failed to build statement parser: " + err.Error())
	}
	return p
}

// newForRangeParser builds a participle parser for parsedForRange.
// This is used for unit testing the for-range grammar in isolation.
func newForRangeParser() *participle.Parser[parsedForRange] {
	p, err := participle.Build[parsedForRange](
		participle.Lexer(buildLexer()),
		participle.Unquote("String"),
		participle.Elide("whitespace"),
		participle.UseLookahead(participle.MaxLookahead),
	)
	if err != nil {
		panic("failed to build for-range parser: " + err.Error())
	}
	return p
}
