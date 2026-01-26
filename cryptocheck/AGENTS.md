# cryptocheck Analyzers

This module contains static analyzers for Go cryptographic code, built on `golang.org/x/tools/go/analysis`.

## Adding a New Analyzer

1. **Create the analyzer package** at `passes/<name>/<name>.go`:
   ```go
   package name

   import (
       "golang.org/x/tools/go/analysis"
       "golang.org/x/tools/go/analysis/passes/inspect"
       "golang.org/x/tools/go/ast/inspector"
   )

   const Doc = `short description

   Longer explanation of what the analyzer checks and why.`

   var Analyzer = &analysis.Analyzer{
       Name:     "name",
       Doc:      Doc,
       URL:      "https://pkg.go.dev/filippo.io/mostly-harmless/cryptocheck/passes/name",
       Requires: []*analysis.Analyzer{inspect.Analyzer},
       Run:      run,
   }

   func run(pass *analysis.Pass) (any, error) {
       inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
       // ... implementation
       return nil, nil
   }
   ```

2. **Create the singlechecker command** at `passes/<name>/cmd/<name>/main.go`:
   ```go
   package main

   import (
       "filippo.io/mostly-harmless/cryptocheck/passes/name"
       "golang.org/x/tools/go/analysis/singlechecker"
   )

   func main() { singlechecker.Main(name.Analyzer) }
   ```

3. **Create the test** at `passes/<name>/<name>_test.go`:
   ```go
   package name_test

   import (
       "testing"
       "filippo.io/mostly-harmless/cryptocheck/passes/name"
       "golang.org/x/tools/go/analysis/analysistest"
   )

   func Test(t *testing.T) {
       testdata := analysistest.TestData()
       analysistest.Run(t, testdata, name.Analyzer, "a")
   }
   ```

4. **Create test cases** at `passes/<name>/testdata/src/a/a.go`:
   - Use `// want "regex"` comments to assert expected diagnostics
   - The regex must match the diagnostic message
   - Include both positive cases (should flag) and negative cases (should not flag)

## Common Patterns

### Using the inspect analyzer with stack

When you need to know the enclosing function or other context:

```go
inspect.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
    if !push {
        return true
    }
    // stack[0] is the *ast.File, stack[len(stack)-1] is n
    // ...
    return true
})
```

### Skipping test files

```go
filename := pass.Fset.Position(n.Pos()).Filename
if strings.HasSuffix(filename, "_test.go") {
    return true // skip
}
```

### Allowing VarTime functions

Many analyzers should skip functions with a `VarTime` suffix in their name:

```go
func inVarTimeFunc(stack []ast.Node) bool {
    for i := len(stack) - 1; i >= 0; i-- {
        if fn, ok := stack[i].(*ast.FuncDecl); ok {
            return strings.HasSuffix(fn.Name.Name, "VarTime")
        }
    }
    return false
}
```

## Documentation

- [go/analysis package](https://pkg.go.dev/golang.org/x/tools/go/analysis)
- [analysistest package](https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest)
- [ast/inspector package](https://pkg.go.dev/golang.org/x/tools/go/ast/inspector)
