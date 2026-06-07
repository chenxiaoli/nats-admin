package api

import (
	"io/fs"

	"github.com/chenxiaoli/nats-admin/internal/api/handler"
	"github.com/chenxiaoli/nats-admin/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Pool       *pgxpool.Pool
	JWTSecret  []byte
	Auth       *handler.AuthHandler
	Tenants    *handler.TenantsHandler
	Creds      *handler.CredentialsHandler
	JS         *handler.JetStreamHandler
	Mon        *handler.MonitorHandler
	FrontendFS fs.FS
}

func NewRouter(d Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Use(middleware.WithAudit(d.Pool))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", d.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(d.JWTSecret))
			r.Post("/auth/refresh", d.Auth.Login)

			r.Route("/tenants", func(r chi.Router) {
				r.Get("/", d.Tenants.List)
				r.Post("/", d.Tenants.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Use(middleware.InjectTenant())
					r.Get("/", d.Tenants.Get)
					r.Put("/", d.Tenants.Update)
					r.Delete("/", d.Tenants.Delete)
					r.Post("/suspend", d.Tenants.Suspend)
					r.Post("/activate", d.Tenants.Activate)
					r.Get("/credentials", d.Creds.List)
					r.Post("/credentials", d.Creds.Issue)
					r.Delete("/credentials/{cid}", d.Creds.Revoke)
					r.Get("/jetstream/streams", d.JS.ListStreams)
					r.Post("/jetstream/streams", d.JS.CreateStream)
					r.Delete("/jetstream/streams/{name}", d.JS.DeleteStream)
					r.Post("/jetstream/streams/{name}/purge", d.JS.PurgeStream)
					r.Get("/jetstream/kv", d.JS.ListKV)
					r.Post("/jetstream/kv", d.JS.CreateKV)
					r.Delete("/jetstream/kv/{bucket}", d.JS.DeleteKV)
				})
			})
			r.Get("/monitor/server", d.Mon.Servers)
			r.Get("/monitor/tenants", d.Mon.Tenants)
			r.Get("/monitor/tenants/{id}", d.Mon.TenantStats)
			r.Get("/ws/monitor", d.Mon.WebSocket)
		})
	})

	// Catch-all for embedded frontend. /api/v1/* and /ws/* above take
	// precedence; anything else falls through to the static handler which
	// serves index.html for unknown paths (SPA client-side routing).
	if d.FrontendFS != nil {
		r.Handle("/*", staticHandler(d.FrontendFS))
	}
	return r
}
