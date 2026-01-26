package nodivision_test

import (
	"testing"

	"filippo.io/mostly-harmless/cryptocheck/passes/nodivision"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nodivision.Analyzer, "a")
}
