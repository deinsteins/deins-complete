package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
)

const maxRequestBodyBytes = 256 * 1024

type CompletionHandler struct {
	service *completion.Service
	logger  *slog.Logger
}

func NewCompletionHandler(service *completion.Service, logger *slog.Logger) *CompletionHandler {
	return &CompletionHandler{service: service, logger: logger}
}

func (handler *CompletionHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestID := middleware.GetRequestID(request.Context())
	if request.ContentLength > maxRequestBodyBytes {
		response.WriteError(writer, http.StatusRequestEntityTooLarge, "INVALID_REQUEST", "Request body is too large.", requestID)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var completionRequest completion.Request
	if err := decoder.Decode(&completionRequest); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.WriteError(writer, http.StatusRequestEntityTooLarge, "INVALID_REQUEST", "Request body is too large.", requestID)
			return
		}
		response.WriteError(writer, http.StatusBadRequest, "INVALID_REQUEST", "Completion request is invalid.", requestID)
		return
	}
	if err := ensureSingleJSONValue(decoder); err != nil || completion.Validate(completionRequest) != nil {
		response.WriteError(writer, http.StatusBadRequest, "INVALID_REQUEST", "Completion request is invalid.", requestID)
		return
	}

	handler.logger.Debug("completion requested", "request_id", requestID, "language", completionRequest.Context.Language, "prefix_length", len(completionRequest.Context.Prefix), "suffix_length", len(completionRequest.Context.Suffix))
	result, err := handler.service.Complete(request.Context(), completionRequest)
	if err != nil {
		if request.Context().Err() != nil {
			return
		}
		handler.logger.Error("completion failed", "request_id", requestID, "error", err)
		response.WriteError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Completion failed.", requestID)
		return
	}
	response.WriteJSON(writer, http.StatusOK, map[string]any{
		"completion": map[string]string{"text": result.Text},
		"metadata":   map[string]string{"requestId": requestID},
	})
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
