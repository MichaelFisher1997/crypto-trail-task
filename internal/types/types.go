package types

import "time"

// GetBalanceRequest represents the incoming payload for balance lookups.
type GetBalanceRequest struct {
	Wallets []string `json:"wallets"`
}

// BalanceEntry represents a single wallet balance response.
type BalanceEntry struct {
	Wallet    string  `json:"wallet"`
	Lamports  uint64  `json:"lamports"`
	Sol       float64 `json:"sol"`
	Source    string  `json:"source"`      // "cache" or "rpc"
	FetchedAt string  `json:"fetched_at"` // RFC3339
}

// ErrorEntry captures per-wallet errors that occurred while fetching.
type ErrorEntry struct {
	Wallet string `json:"wallet"`
	Error  string `json:"error"`
}

// GetBalanceResponse is the JSON response for the balance endpoint.
type GetBalanceResponse struct {
	Balances []BalanceEntry `json:"balances"`
	Errors   []ErrorEntry   `json:"errors"`
}

func NowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// LamportsToSol converts lamports to SOL as a float.
func LamportsToSol(l uint64) float64 { return float64(l) / 1_000_000_000 }

// NewBalanceEntry creates a BalanceEntry from raw lamports and timestamp.
func NewBalanceEntry(wallet string, lamports uint64, source string, ts time.Time) BalanceEntry {
    return BalanceEntry{
        Wallet:    wallet,
        Lamports:  lamports,
        Sol:       LamportsToSol(lamports),
        Source:    source,
        FetchedAt: ts.UTC().Format(time.RFC3339),
    }
}

// SumLamports sums the lamports across a slice of BalanceEntry.
func SumLamports(entries []BalanceEntry) uint64 {
    var total uint64
    for i := range entries {
        total += entries[i].Lamports
    }
    return total
}
