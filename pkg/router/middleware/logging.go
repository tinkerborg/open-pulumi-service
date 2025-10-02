package middleware

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before calling the embedded ResponseWriter's WriteHeader.
func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func Logging(next http.Handler) http.Handler {
	// http.HandlerFunc is an adapter to allow the use of ordinary functions as HTTP handlers.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Initialize the statusRecorder. The default status is 200 OK.
		// If the handler doesn't explicitly call WriteHeader, the status will remain 200.
		rec := &statusRecorder{w, http.StatusOK}

		// Call the next handler in the chain, passing our wrapped ResponseWriter.
		next.ServeHTTP(rec, r)

		// After the handler has completed, the status code has been set (either explicitly
		// or implicitly). We can now access it from our recorder.
		duration := time.Since(start)
		log.Printf("method=%s path=%s status=%d duration=%s",
			r.Method, r.RequestURI, rec.status, duration)
	})
}
