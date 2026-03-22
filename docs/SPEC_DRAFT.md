# OTTL 1.0 Language Specification — DRAFT

> **Status:** draft — created during GSoC 2026 community bonding, not final.

## 1. EBNF Grammar

This grammar covers all existing OTTL constructs plus the proposed `parsedForRange` extension. The grammar is implemented in Go as struct tags using `alecthomas/participle/v2` — there is no separate grammar file. The EBNF below is a human-readable projection of those struct definitions.

```ebnf
(* ============================================ *)
(* Existing OTTL constructs — from grammar.go   *)
(* ============================================ *)

statement           = ( editor | converter ) [ "where" boolean_expression ] ;

editor              = lowercase_name "(" [ argument { "," argument } ] ")" { key } ;
converter           = uppercase_name "(" [ argument { "," argument } ] ")" { key } ;

argument            = [ lowercase_name "=" ] ( value | uppercase_name ) ;

value               = "nil"
                    | math_expr_literal                          (* (?! OpAddSub | OpMultDiv) *)
                    | math_expression
                    | bytes
                    | string
                    | boolean
                    | uppercase_name                              (* enum, (?! Lowercase) *)
                    | map_value
                    | list ;

path                = [ context "." ] field { "." field } ;
context             = lowercase_name ;
field               = lowercase_name { key } ;
key                 = "[" ( string | int | math_expression | math_expr_literal ) "]" ;

list                = "[" { value { "," value } } "]" ;
map_value           = "{" { map_item [ "," ] } "}" ;
map_item            = string ":" value ;

boolean_expression  = term { "or" term } ;
term                = boolean_value { "and" boolean_value } ;
boolean_value       = [ "not" ] ( comparison | const_expr | "(" boolean_expression ")" ) ;
comparison          = value compare_op value ;
compare_op          = "==" | "!=" | ">=" | "<=" | ">" | "<" ;
const_expr          = boolean | converter ;

math_expression     = add_sub_term { ( "+" | "-" ) add_sub_term } ;
add_sub_term        = math_value { ( "*" | "/" ) math_value } ;
math_value          = [ "+" | "-" ] ( math_expr_literal | "(" math_expression ")" ) ;
math_expr_literal   = editor | converter | float | int | path ;

(* ============================================ *)
(* New: for-range extension — GSoC 2026         *)
(* ============================================ *)

for_range           = "for" lowercase_name "," lowercase_name "in" value
                      [ "where" boolean_expression ]
                      "{" { statement } "}" ;

(* TODO: verify that the for-range body { statement } does not conflict with
   map_value { map_item } at the lexer level — current design relies on the
   For token disambiguating before value is attempted. *)
```

## 2. Type Mapping Table

| OTTL Type | Go Type | pdata Type | Zero Value |
|-----------|---------|------------|------------|
| `string` | `string` | `pcommon.ValueTypeStr` | `""` |
| `int` | `int64` | `pcommon.ValueTypeInt` | `0` |
| `float` | `float64` | `pcommon.ValueTypeDouble` | `0.0` |
| `bool` | `bool` | `pcommon.ValueTypeBool` | `false` |
| `bytes` | `[]byte` | `pcommon.ValueTypeBytes` | `[]byte{}` |
| `map` | `pcommon.Map` | `pcommon.ValueTypeMap` | `pcommon.NewMap()` |
| `slice` | `pcommon.Slice` | `pcommon.ValueTypeSlice` | `pcommon.NewSlice()` |
| `nil` | — | `pcommon.ValueTypeEmpty` | — |
| `enum` | `int64` (compile-time) | — | ? |

## 3. Execution Model

An OTTL statement begins its life as a YAML string in a collector configuration file. When the transform processor initialises, it calls `ottl.NewParser[K]()` to build a `participle.Parser[parsedStatement]` backed by the custom lexer from `grammar.go`. Each statement string is parsed into a `parsedStatement` AST node — this is purely syntactic and happens at config-load time, not at runtime. The parser validates syntax but does not yet know whether the function names, path expressions, or argument types are semantically valid.

The parsed AST then enters the compilation phase in `parser.go`. For each `parsedStatement`, the compiler resolves the editor function name against a registry of `Factory[K]` implementations (populated by `ottlfuncs.StandardFuncs` plus any custom functions). It compiles each argument into a `Getter[K]` or `GetSetter[K]` by walking the argument's value tree — paths become `PathExpressionParser` calls that resolve to context-specific accessors, converters become nested function calls, and literals become constant getters. The Where clause, if present, is compiled into a `BoolExpressionEvaluator[K]` using the same recursive descent over the boolean expression tree. The result of compilation is a `Statement[K]` that holds a compiled `ExprFunc[K]` for the editor and an optional `BoolExpressionEvaluator[K]` for the condition.

At runtime, `Statement[K].Execute(ctx, tCtx)` evaluates the condition first — if it returns false, execution short-circuits immediately with `(nil, false, nil)`. If the condition passes (or is absent), the compiled `ExprFunc[K]` is invoked with the same `(ctx, tCtx)` pair, performing the actual mutation on the telemetry data held in `tCtx`. The `StatementSequence[K]` type wraps multiple statements and executes them in declaration order, with error handling controlled by `error_mode` (propagate, ignore, or silent).

## 4. Error Taxonomy

- **Parse error:** the input string does not match the OTTL grammar. Raised at config-load time by participle. TODO: add example messages.
- **Type error:** a semantic mismatch between expected and actual argument types, detectable at compile time if `ottl.StrictTypes` is enabled. TODO: add example messages.
- **Path error:** a path expression that does not map to any known field in the selected context package. Raised during compilation when `PathExpressionParser` returns an error. TODO: add example messages.
- **Runtime error:** any error that occurs during `ExprFunc` evaluation — division by zero, regex compilation failure, conversion overflow. Handled according to `error_mode`. TODO: add example messages.
