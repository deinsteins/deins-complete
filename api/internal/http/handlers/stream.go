package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
)

type StreamHandler struct {
	service *completion.Service
	logger  *slog.Logger
}

func NewStreamHandler(service *completion.Service, logger *slog.Logger) *StreamHandler {
	return &StreamHandler{service: service, logger: logger}
}

func (handler *StreamHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestID := middleware.GetRequestID(request.Context())
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var completionRequest completion.Request
	if err := decoder.Decode(&completionRequest); err != nil || ensureSingleJSONValue(decoder) != nil || completion.Validate(completionRequest) != nil {
		response.WriteError(writer, http.StatusBadRequest, "INVALID_REQUEST", "Completion request is invalid.", requestID)
		return
	}
	if entitlements, ok := middleware.EntitlementsFromContext(request.Context()); ok {
		if !entitlements.Streaming {
			response.WriteError(writer, http.StatusForbidden, "FEATURE_NOT_AVAILABLE", "Streaming is not available for this plan.", requestID)
			return
		}
		if !entitlements.RepositoryContext {
			completionRequest.RepositoryContext = nil
		}
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		flusher = noopFlusher{}
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	result, err := handler.service.Stream(request.Context(), completionRequest, func(chunk string) error {
		return writeEvent(writer, flusher, "chunk", map[string]string{"text": chunk})
	})
	if err != nil {
		if request.Context().Err() == nil {
			handler.logger.Debug("stream completion failed", "request_id", requestID)
			_ = writeEvent(writer, flusher, "error", map[string]string{"code": "PROVIDER_UNAVAILABLE", "requestId": requestID})
		}
		return
	}
	_ = writeEvent(writer, flusher, "done", map[string]string{"text": result.Text, "requestId": requestID})
}

type noopFlusher struct{}

func (noopFlusher) Flush() {}

func writeEvent(writer http.ResponseWriter, flusher http.Flusher, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = writer.Write([]byte("event: " + event + "\ndata: " + string(data) + "\n\n")); err == nil {
		flusher.Flush()
	}
	return err
}
