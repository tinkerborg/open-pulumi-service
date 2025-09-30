package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w ResponseWriter) JSON(response any, errors ...error) error {
	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(response)
	// if err := json.NewEncoder(&buf).Encode(response); err != nil {
	// w.WriteHeader(http.StatusInternalServerError)
	// json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "Failed to encode response"})
	// }

	// return w.Write(buf.Bytes())
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(w.statusCode)
}

func (w ResponseWriter) WithStatus(statusCode int) ResponseWriter {
	w.WriteHeader(statusCode)
	return w
}

// HANDLE NIL ERRORS
func (w ResponseWriter) Error(err error) error {
	if w.statusCode == 0 {
		if err == nil {
			return nil
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	if w.statusCode < 400 {
		return nil
	}

	if err == nil {
		err = errors.New(http.StatusText(w.statusCode))
	}

	return json.NewEncoder(w).Encode(HTTPError{Code: w.statusCode, Message: err.Error()})
}

func (w ResponseWriter) Errorf(format string, a ...any) error {
	return w.Error(fmt.Errorf(format, a...))
}

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
