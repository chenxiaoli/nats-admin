package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/chenxiaoli/nats-admin/internal/api"
	"github.com/chenxiaoli/nats-admin/internal/api/handler"
	"github.com/chenxiaoli/nats-admin/internal/config"
	"github.com/chenxiaoli/nats-admin/internal/credential"
	"github.com/chenxiaoli/nats-admin/internal/db"
	"github.com/chenxiaoli/nats-admin/internal/jetstream"
	"github.com/chenxiaoli/nats-admin/internal/monitor"
	"github.com/chenxiaoli/nats-admin/internal/operator"
	"github.com/chenxiaoli/nats-admin/internal/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/crypto/bcrypt"
)

func runServer() error {
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

	if err := seedAdmin(ctx, pool, cfg); err != nil {
		log.Printf("warn: seed admin: %v", err)
	}

	op, err := operator.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("operator: %w", err)
	}

	if cfg.ResolverDir != "" {
		if err := seedSystemAccountJWT(cfg, op); err != nil {
			log.Printf("warn: seed system account jwt: %v", err)
		}
	}

	var sysConn *nats.Conn
	var res *tenant.Resolver
	sysConn, err = connectSystemAccount(cfg, op)
	if err != nil {
		log.Printf("warn: system account connection failed: %v (resolver/monitor unavailable)", err)
	} else {
		res = tenant.NewResolver(sysConn)
		if err := pushSystemAccountJWT(ctx, cfg, op, res); err != nil {
			log.Printf("warn: push system account jwt to cluster: %v", err)
		}
	}

	tenantSvc := tenant.NewService(tenant.NewRepository(pool), res, op, cfg.MasterKey)
	credSvc := credential.NewService(credential.NewPgRepository(pool), tenantSvc, op, cfg.MasterKey)

	if res != nil {
		if pushed, failed, err := tenantSvc.ReconcileAll(ctx); err != nil {
			log.Printf("warn: reconcile failed: %v", err)
		} else {
			log.Printf("reconcile: pushed %d tenant jwt(s) to resolver (%d skipped/failed)", pushed, failed)
		}
	}

	jsMgr := jetstream.NewManager(cfg.NATSURL)
	defer jsMgr.Close()
	jsAdmin := jetstream.NewAdmin(jsMgr, func(ctx context.Context, tenantID uuid.UUID) (string, string, error) {
		akp, err := tenantSvc.AccountKeyPair(ctx, tenantID)
		if err != nil {
			return "", "", err
		}
		userJWT, userSeed, err := mintBackendUser(akp, "admin")
		if err != nil {
			return "", "", err
		}
		return userJWT, userSeed, nil
	})

	mon := monitor.NewMonitor(sysConn)
	monHub := monitor.NewHub(mon)
	go monHub.Start()
	defer monHub.Stop()
	defer mon.Stop()
	if err := mon.Start(ctx); err != nil {
		log.Printf("warn: monitor start failed: %v", err)
	}

	router := api.NewRouter(api.Deps{
		Pool:       pool,
		JWTSecret:  cfg.JWTSecret,
		Auth:       handler.NewAuthHandler(pool, cfg.JWTSecret, cfg.JWTExpiry),
		Tenants:    handler.NewTenantsHandler(tenantSvc),
		Creds:      handler.NewCredentialsHandler(credSvc),
		JS:         handler.NewJetStreamHandler(jsAdmin),
		Mon:        handler.NewMonitorHandler(mon, monHub),
		FrontendFS: mustSub(frontendFS, "frontend_dist"),
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

func connectSystemAccount(cfg *config.Config, op *operator.Operator) (*nats.Conn, error) {
	sakp, err := nkeys.FromSeed([]byte(cfg.SysAccountSeed))
	if err != nil {
		return nil, fmt.Errorf("parse sys account seed: %w", err)
	}
	saPub, _ := sakp.PublicKey()

	ukp, err := nkeys.CreateUser()
	if err != nil {
		return nil, fmt.Errorf("create sys user nkey: %w", err)
	}
	uPub, _ := ukp.PublicKey()
	uSeed, _ := ukp.Seed()

	claims := jwt.NewUserClaims(uPub)
	claims.Name = "sys"
	claims.IssuerAccount = saPub
	claims.Permissions = jwt.Permissions{
		Pub: jwt.Permission{Allow: []string{">"}},
		Sub: jwt.Permission{Allow: []string{">"}},
	}
	sysUserJWT, err := claims.Encode(sakp)
	if err != nil {
		return nil, fmt.Errorf("sign sys user jwt: %w", err)
	}

	return nats.Connect(cfg.NATSURL,
		nats.UserJWTAndSeed(sysUserJWT, string(uSeed)),
	)
}

func seedSystemAccountJWT(cfg *config.Config, op *operator.Operator) error {
	sakp, err := nkeys.FromSeed([]byte(cfg.SysAccountSeed))
	if err != nil {
		return fmt.Errorf("parse sys account seed: %w", err)
	}
	saPub, _ := sakp.PublicKey()

	claims := jwt.NewAccountClaims(saPub)
	claims.Name = "SYS"
	claims.DefaultPermissions = jwt.Permissions{
		Pub: jwt.Permission{Allow: []string{">"}},
		Sub: jwt.Permission{Allow: []string{">"}},
	}
	sysJWT, err := op.SignAccountClaims(claims)
	if err != nil {
		return fmt.Errorf("sign system account jwt: %w", err)
	}

	if err := os.MkdirAll(cfg.ResolverDir, 0o755); err != nil {
		return fmt.Errorf("mkdir resolver dir: %w", err)
	}
	path := filepath.Join(cfg.ResolverDir, saPub+".jwt")
	existing, readErr := os.ReadFile(path)
	if readErr == nil && string(existing) == sysJWT {
		return nil
	}
	if err := os.WriteFile(path, []byte(sysJWT), 0o644); err != nil {
		return fmt.Errorf("write system jwt: %w", err)
	}
	log.Printf("seeded system account jwt → %s", path)
	return nil
}

func pushSystemAccountJWT(ctx context.Context, cfg *config.Config, op *operator.Operator, res *tenant.Resolver) error {
	sakp, err := nkeys.FromSeed([]byte(cfg.SysAccountSeed))
	if err != nil {
		return fmt.Errorf("parse sys account seed: %w", err)
	}
	saPub, _ := sakp.PublicKey()

	claims := jwt.NewAccountClaims(saPub)
	claims.Name = "SYS"
	claims.DefaultPermissions = jwt.Permissions{
		Pub: jwt.Permission{Allow: []string{">"}},
		Sub: jwt.Permission{Allow: []string{">"}},
	}
	sysJWT, err := op.SignAccountClaims(claims)
	if err != nil {
		return fmt.Errorf("sign system account jwt: %w", err)
	}

	if err := res.Push(ctx, sysJWT); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	log.Printf("pushed system account jwt (%s) to cluster", saPub)
	return nil
}

func mintBackendUser(akp nkeys.KeyPair, name string) (string, string, error) {
	ukp, err := nkeys.CreateUser()
	if err != nil {
		return "", "", fmt.Errorf("create user nkey: %w", err)
	}
	uPub, _ := ukp.PublicKey()
	uSeed, _ := ukp.Seed()

	apub, _ := akp.PublicKey()
	claims := jwt.NewUserClaims(uPub)
	claims.Name = name
	claims.IssuerAccount = apub
	claims.Permissions = jwt.Permissions{
		Pub: jwt.Permission{Allow: []string{">"}},
		Sub: jwt.Permission{Allow: []string{">"}},
	}
	userJWT, err := claims.Encode(akp)
	if err != nil {
		return "", "", fmt.Errorf("sign user jwt: %w", err)
	}
	return userJWT, string(uSeed), nil
}

func mustSub(fsys embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func seedAdmin(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	if cfg.BootstrapAdminEmail == "" || cfg.BootstrapAdminPassword == "" {
		return nil
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&count); err != nil {
		return fmt.Errorf("check admin_users: %w", err)
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapAdminPassword), 12)
	if err != nil {
		return fmt.Errorf("bcrypt: %w", err)
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO admin_users (email, password_hash, role) VALUES ($1, $2, 'super')`,
		cfg.BootstrapAdminEmail, string(hash))
	if err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}
	log.Printf("bootstrapped admin user: %s", cfg.BootstrapAdminEmail)
	return nil
}
