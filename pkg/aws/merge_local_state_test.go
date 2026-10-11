package aws

import (
	"testing"

	ctypes "github.com/scttfrdmn/prism/pkg/types"
)

// TestMergeLocalStateMetadata_PreservesStorageGB (#720): AWS's instance view carries no root volume
// size, so a refresh must keep the one recorded at launch or the ledger stops accruing EBS storage.
func TestMergeLocalStateMetadata_PreservesStorageGB(t *testing.T) {
	local := &ctypes.State{Instances: map[string]ctypes.Instance{"ws": {Name: "ws", StorageGB: 50}}}

	built := &ctypes.Instance{Name: "ws"}
	mergeLocalStateMetadata(built, "ws", "stopped", local)
	if built.StorageGB != 50 {
		t.Fatalf("StorageGB = %v, want 50 carried from local state", built.StorageGB)
	}

	fresh := &ctypes.Instance{Name: "ws", StorageGB: 80}
	mergeLocalStateMetadata(fresh, "ws", "stopped", local)
	if fresh.StorageGB != 80 {
		t.Fatalf("StorageGB = %v, want the built value 80 kept", fresh.StorageGB)
	}
}
