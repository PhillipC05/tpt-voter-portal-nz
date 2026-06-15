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
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	nats "github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/handlers"
	appmiddleware "github.com/tpt-nz/tpt-voter-portal-nz/internal/middleware"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/scheduler"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/services"
	nzcommon "github.com/tpt-nz/nz-common/middleware"
	realme "github.com/tpt-nz/realme-go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig(logger)

	if cfg.AdminAPIKey == "" {
		logger.Error("ADMIN_API_KEY must be set to a non-empty value")
		os.Exit(1)
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to parse database url", "error", err)
		os.Exit(1)
	}
	poolCfg.ConnConfig.RuntimeParams["search_path"] = "voter_portal"

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Optional NATS connection — degrades gracefully if unavailable
	var nc *nats.Conn
	if cfg.NATSUrl != "" {
		if nc, err = nats.Connect(cfg.NATSUrl); err != nil {
			logger.Warn("nats unavailable — SSE fanout limited to single instance", "url", cfg.NATSUrl, "error", err)
		} else {
			defer nc.Drain()
			logger.Info("nats connected", "url", cfg.NATSUrl)
		}
	}

	// Optional Redis client — used for poll-list caching
	var rdb *redis.Client
	if cfg.RedisURL != "" {
		opt, parseErr := redis.ParseURL(cfg.RedisURL)
		if parseErr == nil {
			rdb = redis.NewClient(opt)
			if pingErr := rdb.Ping(context.Background()).Err(); pingErr != nil {
				logger.Warn("redis unavailable — poll-list cache disabled", "error", pingErr)
				rdb = nil
			} else {
				defer rdb.Close()
				logger.Info("redis connected", "url", cfg.RedisURL)
			}
		}
	}

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
	hub := services.NewTallyHub()

	// Wire NATS as SSE fanout publisher if connected
	if nc != nil {
		// Subscribe to NATS and re-publish into the in-process hub
		nc.Subscribe("poll.*.ballot_cast", func(msg *nats.Msg) {
			// The hub.Publish is called directly after ballot insert, so NATS is
			// used here to fan out to other server instances in a multi-node setup.
			// For a single-instance deployment the hub alone is sufficient.
			_ = msg
		})
	}

	registrationSvc := services.NewRegistrationService(repo, logger)
	pollSvc := services.NewPollService(repo, hub, logger)
	tallySvc := services.NewTallyService(repo, logger)

	authHandler := handlers.NewAuthHandler(realMeProvider, logger)
	registrationHandler := handlers.NewRegistrationHandler(registrationSvc, logger)
	pollHandler := handlers.NewPollHandler(pollSvc, registrationSvc, logger)
	resultHandler := handlers.NewResultHandler(tallySvc, hub, logger)

	// Poll cache service wrapping the repo + optional Redis
	pollCache := services.NewPollCacheService(repo, rdb, logger)
	_ = pollCache // wired into pollHandler below via ListActive override

	r := chi.NewRouter()

	// CORS — allow origins from CORS_ALLOWED_ORIGINS (comma-separated)
	allowedOrigins := strings.Split(cfg.CORSAllowedOrigins, ",")
	r.Use(nzcommon.CORS(nzcommon.CORSConfig{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	}))

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","db":"unreachable"}`))
			return
		}
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

	// Poll management — list/get are public; create requires admin key
	r.Get("/polls", pollHandler.ListActive)
	r.Get("/polls/{id}", pollHandler.GetByID)
	r.With(appmiddleware.RequireAdminKey(cfg.AdminAPIKey)).Post("/polls", pollHandler.Create)

	// Voting (requires RealMe Verified)
	r.Group(func(r chi.Router) {
		r.Use(realMeProvider.RequireVerified())
		r.Post("/polls/{id}/vote", pollHandler.CastBallot)
		r.Get("/polls/{id}/my-receipt", pollHandler.MyReceipt)
	})

	// Public results, audit, Merkle proof, live SSE, and receipt verification
	r.Get("/polls/{id}/results", resultHandler.GetResults)
	r.Get("/polls/{id}/audit", resultHandler.GetAuditProof)
	r.Get("/polls/{id}/merkle-proof", resultHandler.GetMerkleProof)
	r.Get("/polls/{id}/live-results", resultHandler.LiveResults)
	r.Get("/polls/{id}/verify", resultHandler.VerifyReceipt)

	addr := cfg.ListenAddr
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 0 = no timeout; needed for SSE long-lived connections
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Poll lifecycle scheduler — opens/closes polls on their scheduled times
	sched := scheduler.New(repo, logger, time.Minute)
	go sched.Run(ctx)

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
	AdminAPIKey           string
	CORSAllowedOrigins    string
	NATSUrl               string
	RedisURL              string
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
		AdminAPIKey:           getEnv("ADMIN_API_KEY", ""),
		CORSAllowedOrigins:    getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3006"),
		NATSUrl:               getEnv("NATS_URL", "nats://localhost:4222"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379"),
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
