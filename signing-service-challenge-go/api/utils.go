package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/api/apiError"
)

// Response is the generic API response container.
type Response struct {
	Data interface{} `json:"data"`
}

// ErrorResponse is the generic error API response container.
type ErrorResponse struct {
	Errors []string `json:"errors"`
}

// WriteInternalError writes a default internal error message as an HTTP response.
func WriteInternalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
}

// WriteErrorResponse takes an HTTP status code and a slice of errors
// and writes those as an HTTP error response in a structured format.
func WriteErrorResponse(w http.ResponseWriter, code int, errorMessage string, additional ...string) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")

	errorResponse := ErrorResponse{
		Errors: append([]string{errorMessage}, additional...),
	}

	bytes, err := json.Marshal(errorResponse)
	if err != nil {
		WriteInternalError(w)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(bytes)), 10))

	w.Write(bytes)
}

// WriteError tries to write a [apiError.Error] into a response writer, if error is not [apiError.Error]
// it writes internal server error instead.
func WriteError(w http.ResponseWriter, err error) {
	var apiErr apiError.Error
	if errors.As(err, &apiErr) {
		messages := apiErr.Messages()
		WriteErrorResponse(w, apiErr.Code(), messages[0], messages[1:]...)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
}

// WriteAPIResponse takes an HTTP status code and a generic data struct
// and writes those as an HTTP response in a structured format.
func WriteAPIResponse(w http.ResponseWriter, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		w.Header().Set("Content-Type", "application/json")

		response := Response{
			Data: data,
		}

		bytes, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			WriteInternalError(w)
		}
		w.Header().Set("Content-Length", strconv.FormatInt(int64(len(bytes)), 10))

		w.Write(bytes)
	}
}

func ParseBody[T interface{ Validate() error }](ctx context.Context, w http.ResponseWriter, r io.Reader) (T, bool) {
	bodyDecoder := json.NewDecoder(r)
	var dto T
	if err := bodyDecoder.Decode(&dto); err != nil {
		slog.ErrorContext(ctx, "unmarshalling dto", "error", err)
		WriteErrorResponse(w, http.StatusBadRequest, "validation failed", err.Error())
		return dto, false
	}
	if err := dto.Validate(); err != nil {
		slog.ErrorContext(ctx, "validating dto", "error", err)
		WriteErrorResponse(w, http.StatusBadRequest, "validation failed", err.Error())
		return dto, false
	}
	return dto, true
}
