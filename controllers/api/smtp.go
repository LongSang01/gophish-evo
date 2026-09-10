package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// SendingProfiles handles requests for the /api/smtp/ endpoint
func (as *Server) SendingProfiles(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		ss, total, err := models.GetSMTPSummaries(uid, pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ListResponse(w, ss, total, http.StatusOK)
	case http.MethodPost:
		s := models.SMTP{}
		err := json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			ErrorResponse(w, "Invalid request", http.StatusBadRequest)
			return
		}
		_, err = models.GetSMTPByName(s.Name, uid)
		if err != gorm.ErrRecordNotFound {
			ErrorResponse(w, "SMTP name already in use", http.StatusConflict)
			log.Error(err)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = uid
		err = models.PostSMTP(&s)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SuccessResponse(w, s, http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SendingProfile contains functions to handle the GET'ing, DELETE'ing, and PUT'ing
// of a SMTP object
func (as *Server) SendingProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		s, err := models.GetSMTP(id, uid)
		if err != nil {
			ErrorResponse(w, "SMTP not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, s, http.StatusOK)
	case http.MethodDelete:
		err := models.DeleteSMTP(id, uid)
		if err != nil {
			ErrorResponse(w, "Error deleting SMTP", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "SMTP Deleted Successfully", http.StatusOK)
	case http.MethodPut:
		s := models.SMTP{}
		err := json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		if s.Id != id {
			ErrorResponse(w, "/:id and /:smtp_id mismatch", http.StatusBadRequest)
			return
		}
		err = s.Validate()
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = uid
		err = models.PutSMTP(&s)
		if err != nil {
			ErrorResponse(w, "Error updating SMTP", http.StatusInternalServerError)
			return
		}
		SuccessResponse(w, s, http.StatusOK)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
