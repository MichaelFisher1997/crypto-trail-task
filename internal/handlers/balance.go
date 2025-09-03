package handlers

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/example/solapi/internal/cache"
	"github.com/example/solapi/internal/solana"
	"github.com/example/solapi/internal/types"
	sol "github.com/gagliardetto/solana-go"
)

const lamportsPerSOL = 1_000_000_000.0

// BalanceDeps bundles dependencies needed by the handler.
type BalanceDeps struct {
	Cache          *cache.Cache
	Fetcher        solana.BalanceFetcher
	Timeout        time.Duration
	MaxConcurrency int
}

type BalanceHandler struct{ Deps BalanceDeps }

func NewBalanceHandler(deps BalanceDeps) *BalanceHandler { return &BalanceHandler{Deps: deps} }

func dedupe(in []string) []string {
	m := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, w := range in {
		if _, ok := m[w]; ok {
			continue
		}
		m[w] = struct{}{}
		out = append(out, w)
	}
	return out
}

func parsePubkey(s string) (sol.PublicKey, bool) {
	pk, err := sol.PublicKeyFromBase58(s)
	if err != nil {
		return sol.PublicKey{}, false
	}
	return pk, true
}

func (h *BalanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req types.GetBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	if len(req.Wallets) == 0 {
		http.Error(w, `{"error":"wallets required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Wallets) > 100 {
		http.Error(w, `{"error":"too many wallets"}`, http.StatusBadRequest)
		return
	}

	wallets := dedupe(req.Wallets)
	resp := types.GetBalanceResponse{Balances: make([]types.BalanceEntry, 0, len(wallets))}

	// parse & collect invalids
	valid := make([]string, 0, len(wallets))
	for _, wstr := range wallets {
		if _, ok := parsePubkey(wstr); !ok {
			resp.Errors = append(resp.Errors, types.ErrorEntry{Wallet: wstr, Error: "invalid public key"})
			continue
		}
		valid = append(valid, wstr)
	}
	// concurrency control
	sem := make(chan struct{}, h.Deps.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, wstr := range valid {
		wstr := wstr
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			pk, _ := parsePubkey(wstr)
			ctx, cancel := context.WithTimeout(r.Context(), h.Deps.Timeout)
			defer cancel()
			val, source, err := h.Deps.Cache.GetOrFetch(ctx, wstr, func(ctx context.Context) (cache.Value, error) {
				lamports, latency, err := h.Deps.Fetcher.GetBalance(ctx, pk)
				if err != nil {
					return cache.Value{}, err
				}
				// log rpc latency only on miss
				log.Printf("event=rpc_fetch wallet=%s latency_ms=%d", wstr, latency.Milliseconds())
				return cache.Value{Lamports: lamports, FetchedAt: time.Now().UTC()}, nil
			})
			if err != nil {
				mu.Lock()
				resp.Errors = append(resp.Errors, types.ErrorEntry{Wallet: wstr, Error: err.Error()})
				mu.Unlock()
				return
			}
			solAmt := float64(val.Lamports) / lamportsPerSOL
			// avoid -0
			if solAmt == 0 {
				solAmt = 0
			}
			mu.Lock()
			resp.Balances = append(resp.Balances, types.BalanceEntry{
				Wallet:    wstr,
				Lamports:  val.Lamports,
				Sol:       math.Round(solAmt*1e9) / 1e9,
				Source:    source,
				FetchedAt: val.FetchedAt.Format(time.RFC3339),
			})
			mu.Unlock()
			log.Printf("event=balance wallet=%s source=%s", wstr, source)
		}()
	}
	wg.Wait()

	// sort by wallet for deterministic tests
	sort.Slice(resp.Balances, func(i, j int) bool { return resp.Balances[i].Wallet < resp.Balances[j].Wallet })

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
