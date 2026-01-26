// The nodivision command runs the nodivision analyzer.
package main

import (
	"filippo.io/mostly-harmless/cryptocheck/passes/nodivision"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(nodivision.Analyzer) }
