package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func connectTestMongo(t *testing.T) (*mongo.Client, func()) {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("skipping: cannot connect to mongo: %v", err)
	}
	if err := cli.Ping(ctx, nil); err != nil {
		_ = cli.Disconnect(context.Background())
		t.Skipf("skipping: mongo ping failed: %v", err)
	}
	cleanup := func() { _ = cli.Disconnect(context.Background()) }
	return cli, cleanup
}

func TestMongoAPIKeyStore_CreateAndValidate(t *testing.T) {
	cli, done := connectTestMongo(t)
	defer done()
	ctx := context.Background()
	store, err := NewMongoAPIKeyStore(ctx, cli, "solapi_test", 200*time.Millisecond)
	if err != nil { t.Fatalf("new store: %v", err) }
	// ensure clean collection
	_ = store.coll.Drop(ctx)
	// create active key
	key := "test-active-123"
	if err := store.Create(ctx, key, true, "userA"); err != nil { t.Fatalf("create: %v", err) }
	ok, err := store.Validate(ctx, key)
	if err != nil { t.Fatalf("validate err: %v", err) }
	if !ok { t.Fatalf("want ok=true") }
	// second call should be served from cache
	ok2, err := store.Validate(ctx, key)
	if err != nil || !ok2 { t.Fatalf("cached validate err=%v ok=%v", err, ok2) }
}

func TestMongoAPIKeyStore_NegativeCache(t *testing.T) {
	cli, done := connectTestMongo(t)
	defer done()
	ctx := context.Background()
	store, err := NewMongoAPIKeyStore(ctx, cli, "solapi_test", 300*time.Millisecond)
	if err != nil { t.Fatalf("new store: %v", err) }
	_ = store.coll.Drop(ctx)
	missing := "no-such-key"
	ok, err := store.Validate(ctx, missing)
	if err != nil { t.Fatalf("validate err: %v", err) }
	if ok { t.Fatalf("expected false for missing key") }
	// Immediate second call should be served from negative cache (still false)
	ok2, err := store.Validate(ctx, missing)
	if err != nil { t.Fatalf("validate2 err: %v", err) }
	if ok2 { t.Fatalf("expected false from negative cache") }
	// Insert the key now; Create updates the cache to active=true immediately
	if err := store.Create(ctx, missing, true, "owner"); err != nil { t.Fatalf("create later: %v", err) }
	ok3, _ := store.Validate(ctx, missing)
	if !ok3 { t.Fatalf("expected true immediately after Create due to cache update") }
}
