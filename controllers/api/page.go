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

// Pages handles requests for the /api/pages/ endpoint
func (as *Server) Pages(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		ps, total, err := models.GetPageSummaries(uid, pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ListResponse(w, ps, total, http.StatusOK)
	case http.MethodPost:
		p := models.Page{}
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			ErrorResponse(w, "Invalid request", http.StatusBadRequest)
			return
		}
		_, err = models.GetPageByName(p.Name, uid)
		if err != gorm.ErrRecordNotFound {
			ErrorResponse(w, "Page name already in use", http.StatusConflict)
			log.Error(err)
			return
		}
		p.ModifiedDate = time.Now().UTC()
		p.UserId = uid
		err = models.PostPage(&p)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SuccessResponse(w, p, http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Page contains functions to handle the GET'ing, DELETE'ing, and PUT'ing
// of a Page object
func (as *Server) Page(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		p, err := models.GetPage(id, uid)
		if err != nil {
			ErrorResponse(w, "Page not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, p, http.StatusOK)
	case http.MethodDelete:
		err := models.DeletePage(id, uid)
		if err != nil {
			ErrorResponse(w, "Error deleting page", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Page Deleted Successfully", http.StatusOK)
	case http.MethodPut:
		p := models.Page{}
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		if p.Id != id {
			ErrorResponse(w, "/:id and /:page_id mismatch", http.StatusBadRequest)
			return
		}
		p.ModifiedDate = time.Now().UTC()
		p.UserId = uid
		err = models.PutPage(&p)
		if err != nil {
			ErrorResponse(w, "Error updating page: "+err.Error(), http.StatusInternalServerError)
			return
		}
		SuccessResponse(w, p, http.StatusOK)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
