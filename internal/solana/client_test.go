package solana

import (
	"context"
	"testing"
	"time"
)

// We'll just exercise constructor and ensure GetBalance propagates errors when RPC URL is invalid.
func TestClient_NewAndErrorPropagation(t *testing.T) {
	cl := NewClient("http://127.0.0.1:5999", "finalized")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// random pubkey (32 zero bytes is not valid, but this just ensures call executes and returns error)
	var pk [32]byte
	_, _, err := cl.GetBalance(ctx, pk)
	if err == nil {
		t.Fatalf("expected error from RPC call")
	}
}
