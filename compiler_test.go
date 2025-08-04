package solc

import (
	"fmt"
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
