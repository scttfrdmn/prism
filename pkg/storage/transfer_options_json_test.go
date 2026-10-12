package storage

import (
	"encoding/json"
	"testing"
)

// TestTransferOptions_MarshalsWithCallback: TransferOptions travels in the daemon's transfer API, so it
// must marshal even with a ProgressCallback set. Without `json:"-"` on the func field every
// `prism storage upload/download` request failed with "json: unsupported type".
func TestTransferOptions_MarshalsWithCallback(t *testing.T) {
	opts := &TransferOptions{PartSize: 5 << 20, Concurrency: 5, ProgressCallback: func(*TransferProgress) {}}
	data, err := json.Marshal(struct {
		Options *TransferOptions `json:"options"`
	}{opts})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back struct {
		Options TransferOptions `json:"options"`
	}
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Options.PartSize != 5<<20 || back.Options.Concurrency != 5 {
		t.Fatalf("round trip lost options: %+v", back.Options)
	}
}
