// Command server is the Knowhere backend: a single API gateway that
// aggregates Zoho Desk ticket monitoring, Devtron release/deployment tracking,
// customer onboarding status, and Devtron license monitoring.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"knowhere/internal/auth"
	"knowhere/internal/config"
	"knowhere/internal/db"
	"knowhere/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[knowhere] ")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("config: %s", cfg)
	if missing := cfg.MissingIntegrations(); len(missing) > 0 {
		log.Printf("warning: unconfigured integrations: %v (those endpoints return 503)", missing)
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()
	log.Printf("database ready at %s (schema applied)", cfg.DBPath)

	// Authentication: seed the admin user (if configured) and warn if the
	// instance has no users at all — otherwise no one can log in.
	authStore := auth.NewStore(database)
	if err := authStore.SeedAdmin(ctx, cfg.Auth.AdminEmail, cfg.Auth.AdminPassword); err != nil {
		log.Fatalf("seed admin: %v", err)
	}
	if cfg.Auth.AdminEmail != "" {
		log.Printf("admin user ensured: %s", cfg.Auth.AdminEmail)
	}
	if has, _ := authStore.HasAnyUser(ctx); !has {
		log.Printf("warning: no users exist — set ADMIN_EMAIL and ADMIN_PASSWORD to create one; all endpoints will reject login until then")
	}
	authSvc := auth.NewService(authStore, cfg.Auth.Secret, cfg.Auth.TokenTTL, cfg.Auth.CookieSecure)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(cfg, database, authSvc),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
