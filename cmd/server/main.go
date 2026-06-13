// Command server is the HTTP entrypoint for the Online Voter Registration & Polling API.
//
// It configures routes, database, RealMe authentication, and graceful shutdown.
//
// Usage:
//
//	export DATABASE_URL="postgres://tptnz:tptnz_dev@localhost:5432/tptnz?sslmode=disable"
//	go run ./cmd/server
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/handlers"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/services"
	"github.com/tpt-nz/realme-go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig(logger)

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	realMeCfg := realme.Config{
		Environment:       realme.Environment(cfg.RealMeEnvironment),
		CertFile:          cfg.RealMeCertFile,
		KeyFile:           cfg.RealMeKeyFile,
		EntityID:          cfg.RealMeEntityID,
		ACSURL:            cfg.RealMeACSURL,
		IdPMetadataFile:   cfg.RealMeIdPMetadataFile,
		IdPMetadataURL:    cfg.RealMeIdPMetadataURL,
		ForceAuthn:        true,
		AllowIDPInitiated: false,
		SessionCookieName: "tpt-voter-portal-session",
		SessionMaxAge:     1800,
	}

	realMeProvider, err := realme.NewProvider(realMeCfg)
	if err != nil {
		logger.Error("failed to create RealMe provider", "error", err)
		os.Exit(1)
	}

	repo := repository.NewVoterRepository(pool)

	registrationSvc := services.NewRegistrationService(repo, logger)
	pollSvc := services.NewPollService(repo, logger)
	tallySvc := services.NewTallyService(repo, logger)

	authHandler := handlers.NewAuthHandler(realMeProvider, logger)
	registrationHandler := handlers.NewRegistrationHandler(registrationSvc, logger)
	pollHandler := handlers.NewPollHandler(pollSvc, registrationSvc, logger)
	resultHandler := handlers.NewResultHandler(tallySvc, logger)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", authHandler.Login)
		r.Get("/callback", authHandler.Callback)
		r.Get("/logout", authHandler.Logout)
		r.Get("/metadata", authHandler.Metadata)
		r.With(realMeProvider.RequireLogin()).Get("/status", authHandler.Status)
	})

	// Voter registration (requires RealMe Verified identity)
	r.Group(func(r chi.Router) {
		r.Use(realMeProvider.RequireVerified())
		r.Post("/register", registrationHandler.Register)
		r.Get("/register/status", registrationHandler.Status)
	})

	// Poll management (admin: create; public: list/get)
	r.Get("/polls", pollHandler.ListActive)
	r.Get("/polls/{id}", pollHandler.GetByID)
	r.Post("/polls", pollHandler.Create) // TODO: scope to admin role in production

	// Voting (requires RealMe Verified)
	r.Group(func(r chi.Router) {
		r.Use(realMeProvider.RequireVerified())
		r.Post("/polls/{id}/vote", pollHandler.CastBallot)
		r.Get("/polls/{id}/my-receipt", pollHandler.MyReceipt)
	})

	// Public results and audit (no auth required — verifiability is a feature)
	r.Get("/polls/{id}/results", resultHandler.GetResults)
	r.Get("/polls/{id}/audit", resultHandler.GetAuditProof)
	r.Get("/polls/{id}/verify", resultHandler.VerifyReceipt)

	addr := cfg.ListenAddr
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

// Config holds all application configuration.
type Config struct {
	ListenAddr            string
	DatabaseURL           string
	RealMeEnvironment     string
	RealMeCertFile        string
	RealMeKeyFile         string
	RealMeEntityID        string
	RealMeACSURL          string
	RealMeIdPMetadataFile string
	RealMeIdPMetadataURL  string
	CookieDomain          string
}

func loadConfig(logger *slog.Logger) *Config {
	cfg := &Config{
		ListenAddr:            getEnv("LISTEN_ADDR", ":8080"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://tptnz:tptnz_dev@localhost:5432/tptnz?sslmode=disable"),
		RealMeEnvironment:     getEnv("REALME_ENVIRONMENT", "mts"),
		RealMeCertFile:        getEnv("REALME_CERT_FILE", "certs/sp.crt"),
		RealMeKeyFile:         getEnv("REALME_KEY_FILE", "certs/sp.key"),
		RealMeEntityID:        getEnv("REALME_ENTITY_ID", "http://localhost:8080/auth/metadata"),
		RealMeACSURL:          getEnv("REALME_ACS_URL", "http://localhost:8080/auth/callback"),
		RealMeIdPMetadataFile: getEnv("REALME_IDP_METADATA_FILE", ""),
		RealMeIdPMetadataURL:  getEnv("REALME_IDP_METADATA_URL", ""),
		CookieDomain:          getEnv("COOKIE_DOMAIN", ""),
	}

	if cfg.RealMeIdPMetadataFile == "" && cfg.RealMeIdPMetadataURL == "" {
		cfg.RealMeIdPMetadataURL = "http://localhost:8081/metadata"
		logger.Warn("no RealMe IdP metadata configured, using mock IdP URL", "url", cfg.RealMeIdPMetadataURL)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
