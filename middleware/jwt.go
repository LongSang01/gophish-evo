package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gophish/gophish/auth"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// writeJSONError writes a JSON error response. Duplicated here to avoid
// importing the api package (which would create a circular dependency).
func writeJWTError(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
}

// RequireJWT is a middleware that validates JWT tokens for protected routes
func RequireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err != nil {
			writeJWTError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			writeJWTError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Get the user from the database to ensure they still exist and are active
		u, err := models.GetUser(claims.UserID)
		if err != nil {
			writeJWTError(w, http.StatusUnauthorized, "User not found")
			return
		}

		if u.AccountLocked {
			writeJWTError(w, http.StatusForbidden, "Account is locked")
			return
		}

		// Set user information in context
		r = ctx.Set(r, "user", u)
		r = ctx.Set(r, "user_id", u.Id)

		next.ServeHTTP(w, r)
	})
}

// RequireJWTOrAPIKey is a middleware that accepts either JWT token or API key
func RequireJWTOrAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try JWT first
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err == nil {
			claims, err := auth.ValidateToken(tokenString)
			if err == nil {
				u, err := models.GetUser(claims.UserID)
				if err == nil && !u.AccountLocked {
					r = ctx.Set(r, "user", u)
					r = ctx.Set(r, "user_id", u.Id)
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Fall back to API key
		RequireAPIKey(next).ServeHTTP(w, r)
	})
}
