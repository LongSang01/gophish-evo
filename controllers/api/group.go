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

// Groups returns a list of groups if requested via GET.
// If requested via POST, APIGroups creates a new group and returns a reference to it.
func (as *Server) Groups(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		gs, err := models.GetGroupSummaries(uid, pp)
		if err != nil {
			ErrorResponse(w, "No groups found", http.StatusNotFound)
			return
		}
		ListResponse(w, gs.Groups, gs.Total, http.StatusOK)
	case http.MethodPost:
		g := models.Group{}
		err := json.NewDecoder(r.Body).Decode(&g)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		_, err = models.GetGroupByName(g.Name, uid)
		if err != gorm.ErrRecordNotFound {
			ErrorResponse(w, "Group name already in use", http.StatusConflict)
			return
		}
		g.ModifiedDate = time.Now().UTC()
		g.UserId = uid
		err = models.PostGroup(&g)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		SuccessResponse(w, g, http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GroupsSummary returns a summary of the groups owned by the current user.
func (as *Server) GroupsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gs, err := models.GetGroupSummaries(ctx.Get(r, "user_id").(int64), models.PageParams{})
	if err != nil {
		log.Error(err)
		ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	SuccessResponse(w, gs, http.StatusOK)
}

// Group returns details about the requested group.
// If the group is not valid, Group returns null.
func (as *Server) Group(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		var g models.Group
		var err error
		if q.Get("pageNum") != "" || q.Get("pageSize") != "" {
			pp := parsePagination(r)
			g, err = models.GetGroupPaged(id, uid, pp)
		} else {
			g, err = models.GetGroup(id, uid)
		}
		if err != nil {
			ErrorResponse(w, "Group not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, g, http.StatusOK)
	case http.MethodDelete:
		g, err := models.GetGroup(id, uid)
		if err != nil {
			ErrorResponse(w, "Group not found", http.StatusNotFound)
			return
		}
		err = models.DeleteGroup(&g)
		if err != nil {
			ErrorResponse(w, "Error deleting group", http.StatusInternalServerError)
			return
		}
		ActionResponse(w, "Group deleted successfully!", http.StatusOK)
	case http.MethodPut:
		g := models.Group{}
		err := json.NewDecoder(r.Body).Decode(&g)
		if err != nil {
			log.Errorf("error decoding group: %v", err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if g.Id != id {
			ErrorResponse(w, "Error: /:id and group_id mismatch", http.StatusInternalServerError)
			return
		}
		g.ModifiedDate = time.Now().UTC()
		g.UserId = uid
		err = models.PutGroup(&g)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		SuccessResponse(w, g, http.StatusOK)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GroupSummary returns a summary of the groups owned by the current user.
func (as *Server) GroupSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	g, err := models.GetGroupSummary(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		ErrorResponse(w, "Group not found", http.StatusNotFound)
		return
	}
	SuccessResponse(w, g, http.StatusOK)
}
