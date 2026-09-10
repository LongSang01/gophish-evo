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
	switch {
	case r.Method == "GET":
		ErrorResponse(w, "Only POSTs allowed", http.StatusBadRequest)
	case r.Method == "POST":
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
}

// IMAPServer handles requests for the /api/imapserver/ endpoint
func (as *Server) IMAPServer(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ss, err := models.GetIMAP(ctx.Get(r, "user_id").(int64))
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SuccessResponse(w, ss, http.StatusOK)

	// POST: Update database
	case r.Method == "POST":
		im := models.IMAP{}
		err := json.NewDecoder(r.Body).Decode(&im)
		if err != nil {
			ErrorResponse(w, "Invalid data. Please check your IMAP settings.", http.StatusBadRequest)
			return
		}
		im.ModifiedDate = time.Now().UTC()
		im.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PostIMAP(&im, ctx.Get(r, "user_id").(int64))
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Successfully saved IMAP settings.", http.StatusCreated)
	}
}
