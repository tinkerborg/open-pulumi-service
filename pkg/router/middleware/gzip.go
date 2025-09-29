package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
)

// GzipDecodeMiddleware wraps an http.Handler to decompress gzipped request bodies.
func GzipDecode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Error creating gzip reader", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = io.NopCloser(gzReader)  // Replace the request body with the decompressed version
			r.Header.Del("Content-Encoding") // Remove the header to prevent issues with downstream handlers
		}
		next.ServeHTTP(w, r)
	})
}
