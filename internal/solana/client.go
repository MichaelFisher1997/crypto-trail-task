package solana

import (
	"context"
	"time"

	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// BalanceFetcher abstracts fetching balances for a wallet.
type BalanceFetcher interface {
	GetBalance(ctx context.Context, pubkey sol.PublicKey) (lamports uint64, latency time.Duration, err error)
}

type Client struct {
	c          *rpc.Client
	commitment rpc.CommitmentType
}

func NewClient(rpcURL string, commitment string) *Client {
	cm := rpc.CommitmentType(commitment)
	if cm == "" {
		cm = rpc.CommitmentFinalized
	}
	return &Client{c: rpc.New(rpcURL), commitment: cm}
}

func (cl *Client) GetBalance(ctx context.Context, pubkey sol.PublicKey) (uint64, time.Duration, error) {
	start := time.Now()
	res, err := cl.c.GetBalance(ctx, pubkey, cl.commitment)
	lat := time.Since(start)
	if err != nil {
		return 0, lat, err
	}
	return uint64(res.Value), lat, nil
}
