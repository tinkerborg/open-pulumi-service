package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tinkerborg/open-pulumi-service/internal/store"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w ResponseWriter) JSON(response any) error {
	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(response)
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(w.statusCode)
}

func (w ResponseWriter) WithStatus(statusCode int) ResponseWriter {
	w.WriteHeader(statusCode)
	return w
}

func (w ResponseWriter) Error(err error) error {
	if w.statusCode == 0 {
		if errors.Is(err, store.ErrNotFound) {
			return w.WithStatus(http.StatusNotFound).Errorf("not found")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	// TODO - hide 500 error causes
	return json.NewEncoder(w).Encode(HTTPError{Code: w.statusCode, Message: err.Error()})
}

func (w ResponseWriter) Errorf(format string, a ...any) error {
	return w.Error(fmt.Errorf(format, a...))
}

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
