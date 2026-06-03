package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/allen/nats-admin/internal/api"
	"github.com/allen/nats-admin/internal/api/handler"
	"github.com/allen/nats-admin/internal/config"
	"github.com/allen/nats-admin/internal/credential"
	"github.com/allen/nats-admin/internal/db"
	"github.com/allen/nats-admin/internal/operator"
	"github.com/allen/nats-admin/internal/tenant"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	op, err := operator.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("operator: %w", err)
	}

	var res *tenant.Resolver // nil in dev — resolver pushes will fail explicitly
	_ = res

	tenantSvc := tenant.NewService(tenant.NewRepository(pool), res, op, cfg.MasterKey)
	credSvc := credential.NewService(credential.NewPgRepository(pool), tenantSvc, op, cfg.MasterKey)

	router := api.NewRouter(api.Deps{
		Pool:      pool,
		JWTSecret: cfg.JWTSecret,
		Auth:      handler.NewAuthHandler(pool, cfg.JWTSecret, cfg.JWTExpiry),
		Tenants:   handler.NewTenantsHandler(tenantSvc),
		Creds:     handler.NewCredentialsHandler(credSvc),
		JS:        &handler.JetStreamHandler{},
		Mon:       &handler.MonitorHandler{},
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", cfg.Port)
	return srv.ListenAndServe()
}
