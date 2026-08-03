package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
	"github.com/google/uuid"
)

var qualityValue = regexp.MustCompile(`^[a-z0-9][a-z0-9+.#_-]{0,39}$`)

type QualityHandler struct{ repo *account.Repository }

func NewQualityHandler(repo *account.Repository) *QualityHandler { return &QualityHandler{repo: repo} }

func (h *QualityHandler) Record(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var body struct {
		EventID      string `json:"eventId"`
		CompletionID string `json:"completionId"`
		Type         string `json:"type"`
		RequestID    string `json:"requestId"`
		Language     string `json:"language"`
		Framework    string `json:"framework"`
		Focus        string `json:"focus"`
		Mode         string `json:"mode"`
		Source       string `json:"source"`
		LatencyMS    int    `json:"latencyMs"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&body) != nil || !validQualityEvent(body.EventID, body.CompletionID, body.Type, body.RequestID, body.Language, body.Framework, body.Focus, body.Mode, body.Source, body.LatencyMS) {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Quality event is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	installation, ok := middleware.InstallationFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.", middleware.GetRequestID(r.Context()))
		return
	}
	event := account.QualityEvent{ID: body.EventID, CompletionID: body.CompletionID, InstallationID: installation.Installation.ID, EventType: body.Type, ServerRequestID: body.RequestID, Language: body.Language, Framework: body.Framework, Focus: body.Focus, Mode: body.Mode, Source: body.Source, LatencyMS: body.LatencyMS}
	if err := h.repo.RecordQualityEvent(r.Context(), event); err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validQualityEvent(eventID, completionID, eventType, requestID, language, framework, focus, mode, source string, latency int) bool {
	if _, err := uuid.Parse(eventID); err != nil {
		return false
	}
	if _, err := uuid.Parse(completionID); err != nil {
		return false
	}
	if eventType != "shown" && eventType != "accepted" {
		return false
	}
	if len(requestID) > 128 || strings.ContainsAny(requestID, "\r\n") {
		return false
	}
	if !qualityValue.MatchString(language) || !qualityValue.MatchString(framework) || !qualityValue.MatchString(focus) {
		return false
	}
	if mode != "fast" && mode != "full" {
		return false
	}
	if source != "backend" && source != "cache" {
		return false
	}
	return latency >= 0 && latency <= 30000
}
