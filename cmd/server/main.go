package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/solapi/internal/auth"
	"github.com/example/solapi/internal/cache"
	"github.com/example/solapi/internal/config"
	apihttp "github.com/example/solapi/internal/http"
	"github.com/example/solapi/internal/handlers"
	"github.com/example/solapi/internal/rate"
	"github.com/example/solapi/internal/solana"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Load()
	if cfg.HeliusURL == "" {
		log.Println("warning: HELIUS_RPC_URL is empty; server will panic on first RPC call")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("mongo connect error: %v", err)
	}
	defer func() {
		_ = mongoClient.Disconnect(context.Background())
	}()

	store, err := auth.NewMongoAPIKeyStore(ctx, mongoClient, cfg.MongoDB, cfg.KeyCacheTTL)
	if err != nil {
		log.Fatalf("api key store init error: %v", err)
	}

	// deps
	cl := solana.NewClient(cfg.HeliusURL, cfg.SolCommitment)
	c := cache.New(cfg.CacheTTL)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{
		Cache:          c,
		Fetcher:        cl,
		Timeout:        cfg.BalanceTimeout,
		MaxConcurrency: cfg.MaxConcurrency,
	})
	lm := rate.NewLimiterMap(cfg.RateLimitRPM, cfg.RateLimitRPM, 5*time.Minute)
	defer lm.Stop()

	router := apihttp.NewRouter(bh, lm, store)

	// Mount extra endpoints on a parent mux without changing router signature.
	mux := http.NewServeMux()
	if cfg.AdminToken != "" {
		// store is *auth.MongoAPIKeyStore, safe to pass
		admin := handlers.NewAdminHandler(store, cfg.AdminToken)
		mux.Handle("/admin/create-key", apihttp.CORS(admin))
	}
	// Public signup (testing only): issues a key for provided owner/email
	signup := handlers.NewSignupHandler(store)
	mux.Handle("/public/signup", apihttp.CORS(signup))

	// Serve frontend pages using Go templates
	indexTmpl := template.Must(template.ParseFiles("web/templates/index.tmpl"))
	testsTmpl := template.Must(template.ParseFiles("web/templates/tests.tmpl"))

	uiIndex := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := indexTmpl.Execute(w, map[string]any{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	uiTests := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := testsTmpl.Execute(w, map[string]any{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	mux.HandleFunc("/ui", uiIndex)
	mux.HandleFunc("/ui/", uiIndex)
	mux.HandleFunc("/ui/tests", uiTests)

	mux.Handle("/", router)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("shutting down...")
	shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
}
