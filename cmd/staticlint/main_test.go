package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestOsExitChecker(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, OsExitAnalyzer, "main")
}
