package auth

import "testing"

func TestHashPrefix_LengthAndDeterminism(t *testing.T) {
	k := "test-key"
	p1 := HashPrefix(k)
	p2 := HashPrefix(k)
	if len(p1) != 8 { t.Fatalf("len=%d", len(p1)) }
	if p1 != p2 { t.Fatalf("non-deterministic: %s vs %s", p1, p2) }
}
