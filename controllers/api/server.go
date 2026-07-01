package api

import (
	"net/http"

	mid "github.com/gophish/gophish/middleware"
	"github.com/gophish/gophish/middleware/ratelimit"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/worker"
	"github.com/gorilla/mux"
)

// ServerOption is an option to apply to the API server.
type ServerOption func(*Server)

// Server represents the routes and functionality of the Gophish API.
// It's not a server in the traditional sense, in that it isn't started and
// stopped. Rather, it's meant to be used as an http.Handler in the
// AdminServer.
type Server struct {
	handler http.Handler
	worker  worker.Worker
	limiter *ratelimit.PostLimiter
}

// NewServer returns a new instance of the API handler with the provided
// options applied.
func NewServer(options ...ServerOption) *Server {
	defaultWorker, _ := worker.New()
	defaultLimiter := ratelimit.NewPostLimiter()
	as := &Server{
		worker:  defaultWorker,
		limiter: defaultLimiter,
	}
	for _, opt := range options {
		opt(as)
	}
	as.registerRoutes()
	return as
}

// WithWorker is an option that sets the background worker.
func WithWorker(w worker.Worker) ServerOption {
	return func(as *Server) {
		as.worker = w
	}
}

func WithLimiter(limiter *ratelimit.PostLimiter) ServerOption {
	return func(as *Server) {
		as.limiter = limiter
	}
}

func (as *Server) registerRoutes() {
	root := mux.NewRouter()
	root = root.StrictSlash(true)

	// Public auth routes (no authentication required)
	public := root.PathPrefix("/api/").Subrouter()
	public.HandleFunc("/auth/login", as.Login).Methods("POST", "OPTIONS")
	public.HandleFunc("/auth/logout", as.Logout).Methods("POST", "OPTIONS")

	// Protected routes requiring JWT or API key
	protected := root.PathPrefix("/api/").Subrouter()
	protected.Use(mid.RequireJWTOrAPIKey)
	protected.Use(mid.EnforceViewOnly)

	// User info routes
	protected.HandleFunc("/auth/me", as.GetCurrentUser).Methods("GET", "OPTIONS")
	protected.HandleFunc("/auth/change-password", as.ChangePassword).Methods("POST", "OPTIONS")
	protected.HandleFunc("/auth/reset-password-required", as.ResetPasswordRequired).Methods("GET", "OPTIONS")

	// Existing API routes
	protected.HandleFunc("/imap/", as.IMAPServer)
	protected.HandleFunc("/imap/validate", as.IMAPServerValidate)
	protected.HandleFunc("/reset", as.Reset)
	protected.HandleFunc("/campaigns/", as.Campaigns)
	protected.HandleFunc("/campaigns/summary", as.CampaignsSummary)
	protected.HandleFunc("/campaigns/{id:[0-9]+}", as.Campaign)
	protected.HandleFunc("/campaigns/{id:[0-9]+}/results", as.CampaignResults)
	protected.HandleFunc("/campaigns/{id:[0-9]+}/summary", as.CampaignSummary)
	protected.HandleFunc("/campaigns/{id:[0-9]+}/complete", as.CampaignComplete)
	protected.HandleFunc("/campaigns/{id:[0-9]+}/launch", as.CampaignLaunch)
	protected.HandleFunc("/groups/", as.Groups)
	protected.HandleFunc("/groups/summary", as.GroupsSummary)
	protected.HandleFunc("/groups/{id:[0-9]+}", as.Group)
	protected.HandleFunc("/groups/{id:[0-9]+}/summary", as.GroupSummary)
	protected.HandleFunc("/templates/", as.Templates)
	protected.HandleFunc("/templates/{id:[0-9]+}", as.Template)
	protected.HandleFunc("/pages/", as.Pages)
	protected.HandleFunc("/pages/{id:[0-9]+}", as.Page)
	protected.HandleFunc("/smtp/", as.SendingProfiles)
	protected.HandleFunc("/smtp/{id:[0-9]+}", as.SendingProfile)
	protected.HandleFunc("/users/", mid.Use(as.Users, mid.RequirePermission(models.PermissionModifySystem)))
	protected.HandleFunc("/users/{id:[0-9]+}", mid.Use(as.User))
	protected.HandleFunc("/util/send_test_email", as.SendTestEmail)
	protected.HandleFunc("/import/group", as.ImportGroup)
	protected.HandleFunc("/import/email", as.ImportEmail)
	protected.HandleFunc("/import/site", as.ImportSite)
	protected.HandleFunc("/webhooks/", mid.Use(as.Webhooks, mid.RequirePermission(models.PermissionModifySystem)))
	protected.HandleFunc("/webhooks/{id:[0-9]+}/validate", mid.Use(as.ValidateWebhook, mid.RequirePermission(models.PermissionModifySystem)))
	protected.HandleFunc("/webhooks/{id:[0-9]+}", mid.Use(as.Webhook, mid.RequirePermission(models.PermissionModifySystem)))
	as.handler = root
}

func (as *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.handler.ServeHTTP(w, r)
}
