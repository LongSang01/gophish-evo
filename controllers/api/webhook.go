package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/webhook"
	"github.com/gorilla/mux"
)

// Webhooks returns a list of webhooks, both active and disabled
func (as *Server) Webhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		pp := parsePagination(r)
		whs, total, err := models.GetWebhookSummaries(pp)
		if err != nil {
			log.Error(err)
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ListResponse(w, whs, total, http.StatusOK)
	case http.MethodPost:
		wh := models.Webhook{}
		err := json.NewDecoder(r.Body).Decode(&wh)
		if err != nil {
			ErrorResponse(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}
		err = models.PostWebhook(&wh)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		SuccessResponse(w, wh, http.StatusCreated)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Webhook returns details of a single webhook specified by "id" parameter
func (as *Server) Webhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch r.Method {
	case http.MethodGet:
		wh, err := models.GetWebhook(id)
		if err != nil {
			ErrorResponse(w, "Webhook not found", http.StatusNotFound)
			return
		}
		SuccessResponse(w, wh, http.StatusOK)
	case http.MethodDelete:
		err := models.DeleteWebhook(id)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("Deleted webhook with id: %d", id)
		ActionResponse(w, "Webhook deleted Successfully!", http.StatusOK)
	case http.MethodPut:
		wh := models.Webhook{}
		err := json.NewDecoder(r.Body).Decode(&wh)
		if err != nil {
			log.Errorf("error decoding webhook: %v", err)
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		wh.Id = id
		err = models.PutWebhook(&wh)
		if err != nil {
			ErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		SuccessResponse(w, wh, http.StatusOK)
	default:
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ValidateWebhook makes an HTTP request to a specified remote url to ensure that it's valid.
func (as *Server) ValidateWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type validationEvent struct {
		Success bool `json:"success"`
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	wh, err := models.GetWebhook(id)
	if err != nil {
		log.Error(err)
		ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payload := validationEvent{Success: true}
	err = webhook.Send(webhook.EndPoint{URL: wh.URL, Secret: wh.Secret}, payload)
	if err != nil {
		ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}
	SuccessResponse(w, wh, http.StatusOK)
}
