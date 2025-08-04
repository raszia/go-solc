package solc

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	c, err := New(VersionLatest, "./.solc")
	if err != nil {
		t.Fatalf("failed to create compiler: %v", err)
	}
	outputSelection := map[string]map[string][]string{
		"*": {
			"*": {"evm.bytecode.object", "evm.deployedBytecode.object", "abi"},
		},
	}
	contract, err := c.Compile("src", "SimpleStorage",
		outputSelection, WithOptimizer(&Optimizer{Enabled: true, Runs: 999999}),
	)
	if err != nil {
		t.Fatalf("solc compile failed: %v", err)
	}

	for _, c := range contract {
		for name, a := range c {
			fmt.Printf("Contract: %s, ABI: %s\n", name, a.ABI)
			fmt.Printf("Bytecode: %s\n", a.EVM.Bytecode.Object)
			fmt.Printf("Deployed Bytecode: %s\n", a.EVM.DeployedBytecode.Object)

		}
	}
}

// createDummySolc writes a dummy solc executable that simply emits a valid JSON output.
func createDummySolc(t *testing.T, dir string) string {
	dummyPath := filepath.Join(dir, "solc")
	script := `#!/bin/sh
# Read and discard the input then output a minimal valid JSON response.
cat > /dev/null
echo '{"contracts":{}}'
`
	if err := ioutil.WriteFile(dummyPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write dummy solc: %v", err)
	}
	return dummyPath
}

func createDummyContract(t *testing.T, dir, name, content string) {
	// Write a file with .sol extension.
	contractPath := filepath.Join(dir, name+".sol")
	if err := os.WriteFile(contractPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write dummy contract: %v", err)
	}
}

func TestCompileSuccess(t *testing.T) {
	// Create a temporary directory for the dummy solc binary.
	tempSolcDir := t.TempDir()
	dummySolc := createDummySolc(t, tempSolcDir)

	// Create a temporary directory to simulate the source directory with a dummy .sol file.
	tempSrcDir := t.TempDir()
	createDummyContract(t, tempSrcDir, "Dummy", "pragma solidity ^0.8.0;")

	// Create a compiler instance using the dummy solc.
	compiler, err := New(VersionLatest, dummySolc)
	if err != nil {
		t.Fatalf("failed to create compiler: %v", err)
	}

	// Prepare a basic output selection.
	outputSelection := map[string]map[string][]string{
		"*": {
			"*": {"evm.bytecode.object", "evm.deployedBytecode.object", "abi"},
		},
	}

	// Use a simple option: WithOptimizer.
	// Assuming WithOptimizer and Optimizer are defined e.g. in compiler.go.
	compiled, err := compiler.Compile(tempSrcDir, "Dummy", outputSelection, WithOptimizer(&Optimizer{Enabled: true, Runs: 200}))
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	// In our dummy solc response, contracts will be empty.
	if len(compiled) != 0 {
		t.Errorf("Expected 0 contracts, got %v", len(compiled))
	}
}

func TestCompileNonExistentDir(t *testing.T) {
	// Create a temporary dummy solc binary.
	tempSolcDir := t.TempDir()
	dummySolc := createDummySolc(t, tempSolcDir)

	compiler, err := New(VersionLatest, dummySolc)
	if err != nil {
		t.Fatalf("failed to create compiler: %v", err)
	}

	// Use a non-existent directory.
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent")
	outputSelection := map[string]map[string][]string{
		"*": {"*": {"abi"}},
	}

	_, err = compiler.Compile(nonExistentDir, "NoContract", outputSelection, WithOptimizer(&Optimizer{Enabled: false}))
	if err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Errorf("Expected directory error, got: %v", err)
	}
}

func TestCompileInvalidSolc(t *testing.T) {
	// Provide an invalid solc binary path.
	invalidPath := filepath.Join(t.TempDir(), "nosolc")
	compiler, err := New(VersionLatest, invalidPath)
	if err == nil {
		// Create a temporary dummy source directory.
		tempSrcDir := t.TempDir()
		createDummyContract(t, tempSrcDir, "Contract", "pragma solidity ^0.8.0;")
		outputSelection := map[string]map[string][]string{
			"*": {"*": {"abi"}},
		}

		_, err = compiler.Compile(tempSrcDir, "Contract", outputSelection, WithOptimizer(&Optimizer{Enabled: false}))
	}
	if err == nil {
		t.Error("Expected error due to invalid solc binary path, but got none")
	}
}

// Optionally, verify caching behavior by running Compile twice.
func TestCompileCaching(t *testing.T) {
	// Create temporary directory for dummy solc.
	tempSolcDir := t.TempDir()
	dummySolc := createDummySolc(t, tempSolcDir)

	// Create a temporary source directory.
	tempSrcDir := t.TempDir()
	createDummyContract(t, tempSrcDir, "CacheTest", "pragma solidity ^0.8.0;")

	compiler, err := New(VersionLatest, dummySolc)
	if err != nil {
		t.Fatalf("failed to create compiler: %v", err)
	}

	outputSelection := map[string]map[string][]string{
		"*": {"*": {"abi"}},
	}

	// First compilation.
	compiled1, err := compiler.Compile(tempSrcDir, "CacheTest", outputSelection, WithOptimizer(&Optimizer{Enabled: true, Runs: 300}))
	if err != nil {
		t.Fatalf("first Compile failed: %v", err)
	}

	// Second compilation: should hit the cache.
	compiled2, err := compiler.Compile(tempSrcDir, "CacheTest", outputSelection, WithOptimizer(&Optimizer{Enabled: true, Runs: 300}))
	if err != nil {
		t.Fatalf("second Compile failed: %v", err)
	}

	// Since dummy solc returns an empty contract map, both compilations should match.
	if len(compiled1) != len(compiled2) {
		t.Errorf("expected cached result to be equal; got %v and %v", compiled1, compiled2)
	}
}

// To ensure the dummy solc script is used in our tests, we can check that exec.Command runs it.
// This test verifies that exec.Command can call our dummy binary.
func TestDummySolcExecution(t *testing.T) {
	tempSolcDir := t.TempDir()
	dummySolc := createDummySolc(t, tempSolcDir)

	cmd := exec.Command(dummySolc)
	// Provide a minimal standard JSON input.
	cmd.Stdin = strings.NewReader(`{"dummy":"input"}`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dummy solc execution failed: %v", err)
	}
	// Check the output contains our dummy JSON.
	if !strings.Contains(string(out), `"contracts":{}`) {
		t.Errorf("unexpected dummy solc output: %s", out)
	}
}
