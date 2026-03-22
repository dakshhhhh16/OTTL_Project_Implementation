# Backward-Compatibility Test Corpus

These `.ottl` files contain real OTTL statement strings that parse correctly
against the current grammar. They serve as a regression guard: **any change to
the OTTL grammar that causes any statement in these files to parse differently
is a regression that must be fixed before merging.**

## Usage

In the real `pkg/ottl/`, a test function iterates over every `.ottl` file in
this directory, parses each line, and compares the resulting AST against a
cached golden output:

```go
func TestBackwardCompatibility(t *testing.T) {
    files, _ := filepath.Glob("testdata/compat/*.ottl")
    for _, f := range files {
        t.Run(filepath.Base(f), func(t *testing.T) {
            lines := readLines(f)
            for _, line := range lines {
                _, err := parser.ParseStatements([]string{line})
                require.NoError(t, err, "regression: %s: %q", f, line)
            }
        })
    }
}
```

New grammar features (like `parsedForRange`) are validated by NEW corpus files,
not by modifying existing ones.
