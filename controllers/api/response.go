package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/gophish/gophish/logger"
)

// JSONResponse attempts to set the status code, c, and marshal the given interface, d, into a response that
// is written to the given ResponseWriter.
func JSONResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		log.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}

// SuccessResponse writes {success: true, data: ...} for single-object endpoints.
func SuccessResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	JSONResponse(w, map[string]interface{}{
		"success": true,
		"data":    data,
	}, statusCode)
}

// ErrorResponse writes {success: false, message: "..."} for error responses.
func ErrorResponse(w http.ResponseWriter, msg string, statusCode int) {
	JSONResponse(w, map[string]interface{}{
		"success": false,
		"message": msg,
	}, statusCode)
}

// ListResponse writes {success: true, items: [...], total: N} for paginated list endpoints.
func ListResponse(w http.ResponseWriter, items interface{}, total int64, statusCode int) {
	JSONResponse(w, map[string]interface{}{
		"success": true,
		"items":   items,
		"total":   total,
	}, statusCode)
}

// ActionResponse writes {success: true, message: "..."} for action endpoints
// (delete, launch, complete, etc.).
func ActionResponse(w http.ResponseWriter, msg string, statusCode int) {
	JSONResponse(w, map[string]interface{}{
		"success": true,
		"message": msg,
	}, statusCode)
}
