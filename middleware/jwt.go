package middleware

import (
	"net/http"

	"github.com/gophish/gophish/auth"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// RequireJWT is a middleware that validates JWT tokens for protected routes
func RequireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err != nil {
			http.Error(w, `{"success": false, "message": "Authentication required"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, `{"success": false, "message": "Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Get the user from the database to ensure they still exist and are active
		u, err := models.GetUser(claims.UserID)
		if err != nil {
			http.Error(w, `{"success": false, "message": "User not found"}`, http.StatusUnauthorized)
			return
		}

		if u.AccountLocked {
			http.Error(w, `{"success": false, "message": "Account is locked"}`, http.StatusForbidden)
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
