package api

import (
	"encoding/json"
	"net/http"
	"time"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/imap"
	"github.com/gophish/gophish/models"
)

// IMAPServerValidate handles requests for the /api/imapserver/validate endpoint
func (as *Server) IMAPServerValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ErrorResponse(w, "Only POSTs allowed", http.StatusBadRequest)
		return
	}
	im := models.IMAP{}
	err := json.NewDecoder(r.Body).Decode(&im)
	if err != nil {
		ErrorResponse(w, "Invalid request", http.StatusBadRequest)
		return
	}
	err = imap.Validate(&im)
	if err != nil {
		ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}
	ActionResponse(w, "Successful login.", http.StatusCreated)
}

// IMAPServer handles requests for the /api/imapserver/ endpoint
func (as *Server) IMAPServer(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		ss, err := models.GetIMAP(uid)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SuccessResponse(w, ss, http.StatusOK)
	case http.MethodPost:
		im := models.IMAP{}
		err := json.NewDecoder(r.Body).Decode(&im)
		if err != nil {
			ErrorResponse(w, "Invalid data. Please check your IMAP settings.", http.StatusBadRequest)
			return
		}
		im.ModifiedDate = time.Now().UTC()
		im.UserId = uid
		err = models.PostIMAP(&im, uid)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Successfully saved IMAP settings.", http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
