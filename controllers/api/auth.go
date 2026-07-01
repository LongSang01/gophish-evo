package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gophish/gophish/auth"
	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Token   string       `json:"token,omitempty"`
	User    *models.User `json:"user,omitempty"`
}

// ChangePasswordRequest represents the password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// Login handles user authentication and returns a JWT token
func (as *Server) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, LoginResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, LoginResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	// Find the user
	u, err := models.GetUserByUsername(req.Username)
	if err != nil {
		log.Error(err)
		JSONResponse(w, LoginResponse{Success: false, Message: "Invalid username or password"}, http.StatusUnauthorized)
		return
	}

	// Validate password
	if err := auth.ValidatePassword(req.Password, u.Hash); err != nil {
		log.Error(err)
		JSONResponse(w, LoginResponse{Success: false, Message: "Invalid username or password"}, http.StatusUnauthorized)
		return
	}

	// Check if account is locked
	if u.AccountLocked {
		JSONResponse(w, LoginResponse{Success: false, Message: "Account is locked"}, http.StatusForbidden)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(u.Id, u.Username, u.Role.Slug)
	if err != nil {
		log.Error(err)
		JSONResponse(w, LoginResponse{Success: false, Message: "Error generating token"}, http.StatusInternalServerError)
		return
	}

	// Update last login time
	u.LastLogin = time.Now().UTC()
	if err := models.PutUser(&u); err != nil {
		log.Error(err)
	}

	JSONResponse(w, LoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    &u,
	}, http.StatusOK)
}

// Logout handles user logout
func (as *Server) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Logged out successfully"}, http.StatusOK)
}

// ChangePassword handles password change requests
func (as *Server) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	u := ctx.Get(r, "user").(models.User)

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	// Validate current password
	if err := auth.ValidatePassword(req.CurrentPassword, u.Hash); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Current password is incorrect"}, http.StatusBadRequest)
		return
	}

	// Validate and get new password hash
	newHash, err := auth.ValidatePasswordChange(u.Hash, req.NewPassword, req.ConfirmPassword)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Update password
	u.Hash = string(newHash)
	u.PasswordChangeRequired = false
	if err := models.PutUser(&u); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error updating password"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Password changed successfully"}, http.StatusOK)
}

// GetCurrentUser returns the currently authenticated user
func (as *Server) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	u := ctx.Get(r, "user").(models.User)
	JSONResponse(w, u, http.StatusOK)
}

// ResetPasswordRequired checks if the current user needs to reset their password
func (as *Server) ResetPasswordRequired(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	u := ctx.Get(r, "user").(models.User)
	JSONResponse(w, map[string]bool{
		"password_change_required": u.PasswordChangeRequired,
	}, http.StatusOK)
}
