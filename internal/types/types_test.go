package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNowRFC3339_Format(t *testing.T) {
	v := NowRFC3339()
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		t.Fatalf("not RFC3339: %v", err)
	}
}

func TestGetBalanceResponse_JSON(t *testing.T) {
	resp := GetBalanceResponse{
		Balances: []BalanceEntry{{Wallet: "w1", Lamports: 123, Sol: 0.00123, Source: "cache", FetchedAt: NowRFC3339()}},
		Errors:   []ErrorEntry{{Wallet: "w2", Error: "bad"}},
	}
	b, err := json.Marshal(resp)
	if err != nil { t.Fatalf("marshal: %v", err) }
	var back GetBalanceResponse
	if err := json.Unmarshal(b, &back); err != nil { t.Fatalf("unmarshal: %v", err) }
	if len(back.Balances) != 1 || back.Balances[0].Wallet != "w1" { t.Fatalf("unexpected decode: %+v", back) }
}

func TestLamportsToSol(t *testing.T) {
	got := LamportsToSol(2_000_000_000)
	if got != 2.0 { t.Fatalf("want 2.0 got %v", got) }
}

func TestNewBalanceEntry(t *testing.T) {
	ts := time.Now()
	be := NewBalanceEntry("w", 1_500_000_000, "rpc", ts)
	if be.Wallet != "w" || be.Sol != 1.5 || be.Source != "rpc" {
		t.Fatalf("bad entry: %+v", be)
	}
}

func TestSumLamports(t *testing.T) {
	entries := []BalanceEntry{
		{Lamports: 1}, {Lamports: 2}, {Lamports: 3},
	}
	if got := SumLamports(entries); got != 6 {
		t.Fatalf("sum=%d", got)
	}
}
