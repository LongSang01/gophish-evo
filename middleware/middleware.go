package middleware

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// Use allows us to stack middleware to process the request
// Example taken from https://github.com/gorilla/mux/pull/36#issuecomment-25849172
func Use(handler http.HandlerFunc, mid ...func(http.Handler) http.HandlerFunc) http.HandlerFunc {
	for _, m := range mid {
		handler = m(handler)
	}
	return handler
}

// GetContext wraps each request in a function which fills in the context for a given request.
// This is kept for backward compatibility but now only sets basic context.
func GetContext(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the request form
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing request", http.StatusInternalServerError)
		}
		handler.ServeHTTP(w, r)
		// Remove context contents
		ctx.Clear(r)
	}
}

// RequireAPIKey ensures that a valid API key is set as either the api_key GET
// parameter, or a Bearer token.
func RequireAPIKey(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		ak := r.Form.Get("api_key")
		// If we can't get the API key, we'll also check for the
		// Authorization Bearer token
		if ak == "" {
			ak = r.Header.Get("Authorization")
			if len(ak) > 7 && ak[:7] == "Bearer " {
				ak = ak[7:]
			}
		}
		if ak == "" {
			writeJSONError(w, http.StatusUnauthorized, "API Key not set")
			return
		}
		u, err := models.GetUserByAPIKey(ak)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Invalid API Key")
			return
		}
		r = ctx.Set(r, "user", u)
		r = ctx.Set(r, "user_id", u.Id)
		r = ctx.Set(r, "api_key", ak)
		handler.ServeHTTP(w, r)
	})
}

// EnforceViewOnly is a global middleware that limits the ability to edit
// objects to accounts with the PermissionModifyObjects permission.
func EnforceViewOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			user, ok := ctx.Get(r, "user").(models.User)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "Authentication required")
				return
			}
			access, err := user.HasPermission(models.PermissionModifyObjects)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if !access {
				writeJSONError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePermission is a middleware that ensures the user has the given
// permission before executing the handler. If the user does not have the
// permission, a JSON 403 Forbidden response is returned.
func RequirePermission(perm string) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := ctx.Get(r, "user").(models.User)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "Authentication required")
				return
			}
			access, err := user.HasPermission(perm)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if !access {
				writeJSONError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// ApplySecurityHeaders applies various security headers according to best-
// practices.
func ApplySecurityHeaders(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		csp := "frame-ancestors 'none';"
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	}
}

// writeJSONError writes a JSON error response with the given status code and message.
func writeJSONError(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
}
