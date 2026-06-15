package handlers

import (
	"log/slog"
	"net/http"

	"github.com/tpt-nz/realme-go"
)

// AuthHandler groups the RealMe authentication HTTP handlers.
type AuthHandler struct {
	provider *realme.Provider
	logger   *slog.Logger
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(provider *realme.Provider, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		provider: provider,
		logger:   logger.With("handler", "auth"),
	}
}

// Login initiates the RealMe login flow.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.provider.LoginHandler()(w, r)
}

// Callback handles the RealMe SAML callback after authentication.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	h.provider.CallbackHandler(nil)(w, r)
}

// Logout handles user logout and session destruction.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.provider.LogoutHandler()(w, r)
}

// Metadata serves the SAML metadata XML for RealMe service registration.
func (h *AuthHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	h.provider.MetadataHandler()(w, r)
}

// Status returns the current authentication status.
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	identity := realme.IdentityFromContext(r.Context())
	if identity == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"identity": map[string]interface{}{
			"fullName":   identity.FullName,
			"assurance":  identity.AssuranceLevel.String(),
			"isVerified": identity.IsVerified(),
		},
	})
}
