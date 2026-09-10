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

// Templates handles the functionality for the /api/templates endpoint
func (as *Server) Templates(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		ts, total, err := models.GetTemplateSummaries(uid, pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ListResponse(w, ts, total, http.StatusOK)
	case http.MethodPost:
		t := models.Template{}
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		_, err = models.GetTemplateByName(t.Name, uid)
		if err != gorm.ErrRecordNotFound {
			ErrorResponse(w, "Template name already in use", http.StatusConflict)
			return
		}
		t.ModifiedDate = time.Now().UTC()
		t.UserId = uid
		err = models.PostTemplate(&t)
		if err == models.ErrTemplateNameNotSpecified {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err == models.ErrTemplateMissingParameter {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			ErrorResponse(w, "Error inserting template into database", http.StatusInternalServerError)
			log.Error(err)
			return
		}
		SuccessResponse(w, t, http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Template handles the functions for the /api/templates/:id endpoint
func (as *Server) Template(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		t, err := models.GetTemplate(id, uid)
		if err != nil {
			ErrorResponse(w, "Template not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, t, http.StatusOK)
	case http.MethodDelete:
		err := models.DeleteTemplate(id, uid)
		if err != nil {
			ErrorResponse(w, "Error deleting template", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Template deleted successfully!", http.StatusOK)
	case http.MethodPut:
		t := models.Template{}
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		if t.Id != id {
			ErrorResponse(w, "Error: /:id and template_id mismatch", http.StatusBadRequest)
			return
		}
		t.ModifiedDate = time.Now().UTC()
		t.UserId = uid
		err = models.PutTemplate(&t)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		SuccessResponse(w, t, http.StatusOK)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
