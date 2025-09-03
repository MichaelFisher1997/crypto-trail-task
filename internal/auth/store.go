package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// APIKeyStore validates API keys and optionally provides a health ping.
type APIKeyStore interface {
	Validate(ctx context.Context, key string) (bool, error)
	Ping(ctx context.Context) error
}

// APIKeyCreator exposes creation/upsert of API keys for admin/signup handlers.
// Implemented by MongoAPIKeyStore.
type APIKeyCreator interface {
	Create(ctx context.Context, key string, active bool, owner string) error
}

type cacheEntry struct {
	active    bool
	expiresAt time.Time
}

type MongoAPIKeyStore struct {
	coll       *mongo.Collection
	cacheTTL   time.Duration
	mu         sync.RWMutex
	cache      map[string]cacheEntry
}

type apiKeyDoc struct {
	Key    string `bson:"key"`
	Active bool   `bson:"active"`
	Owner  string `bson:"owner,omitempty"`
}

// NewMongoAPIKeyStore sets up the collection and unique index on key.
func NewMongoAPIKeyStore(ctx context.Context, client *mongo.Client, dbName string, ttl time.Duration) (*MongoAPIKeyStore, error) {
	coll := client.Database(dbName).Collection("api_keys")
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}
	return &MongoAPIKeyStore{
		coll:     coll,
		cacheTTL: ttl,
		cache:    make(map[string]cacheEntry),
	}, nil
}

func (s *MongoAPIKeyStore) Validate(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, errors.New("missing key")
	}
	// check cache first
	s.mu.RLock()
	ce, ok := s.cache[key]
	s.mu.RUnlock()
	if ok && time.Now().Before(ce.expiresAt) {
		return ce.active, nil
	}
	// fetch from DB
	var doc apiKeyDoc
	err := s.coll.FindOne(ctx, bson.D{{Key: "key", Value: key}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// cache negative result briefly to avoid hammering DB
			s.mu.Lock()
			s.cache[key] = cacheEntry{active: false, expiresAt: time.Now().Add(s.cacheTTL)}
			s.mu.Unlock()
			return false, nil
		}
		return false, err
	}
	active := doc.Active
	s.mu.Lock()
	s.cache[key] = cacheEntry{active: active, expiresAt: time.Now().Add(s.cacheTTL)}
	s.mu.Unlock()
	return active, nil
}

func (s *MongoAPIKeyStore) Ping(ctx context.Context) error {
	return s.coll.Database().Client().Ping(ctx, nil)
}

// Create inserts or updates an API key entry. This is not required by the APIKeyStore interface,
// but available on the Mongo implementation for admin workflows.
func (s *MongoAPIKeyStore) Create(ctx context.Context, key string, active bool, owner string) error {
	if key == "" {
		return errors.New("missing key")
	}
	_, err := s.coll.UpdateOne(ctx,
		bson.D{{Key: "key", Value: key}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "active", Value: active}, {Key: "owner", Value: owner}}}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}
	// update cache
	s.mu.Lock()
	s.cache[key] = cacheEntry{active: active, expiresAt: time.Now().Add(s.cacheTTL)}
	s.mu.Unlock()
	return nil
}

// HashPrefix returns the first 8 hex chars of SHA-256(key) for logging.
func HashPrefix(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:8]
}
