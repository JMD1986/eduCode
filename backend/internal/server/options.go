package server

import "time"

// Options configures CORS, optional OAuth login, and unsafe dev authentication.
type Options struct {
	AllowedOrigins []string

	// AuthDev enables the legacy X-User-Subject header for local tooling/tests only.
	// Never enable in production.
	AuthDev bool

	// OAuth enables Authorization Code + PKCE flows under /api/auth when non-nil.
	OAuth *OAuthSettings
}

// OAuthSettings is loaded from environment (secrets belong in Secret Manager in GCP).
type OAuthSettings struct {
	Issuer   string
	ClientID string
	// ClientSecret is used only on the token endpoint from this server (never exposed to the browser).
	ClientSecret string

	// RedirectURL must exactly match the redirect_uri registered at the IdP (typically proxied via Vite in dev).
	RedirectURL string

	// AppPublicOrigin is where users land after login (browser-visible origin, e.g. http://localhost:5173).
	AppPublicOrigin string

	Scopes []string

	// CookieSecure sets the Secure flag (required for HTTPS deployments).
	CookieSecure bool

	SessionTTL time.Duration
}
