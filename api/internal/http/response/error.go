package response

import "net/http"

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func WriteError(writer http.ResponseWriter, status int, code, message, requestID string) {
	WriteJSON(writer, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message, RequestID: requestID}})
}
