package middleware

import (
	"context"
	"net/http"
	"path"
)

type dynamicPrefixKey struct{}

type DynamicPrefix[T any] struct {
	prefix string
	parser func(r *http.Request) (T, error)
}

func NewDynamicPrefix[T any](prefix string, parser func(r *http.Request) (T, error)) *DynamicPrefix[T] {
	return &DynamicPrefix[T]{prefix, parser}
}

func (d *DynamicPrefix[T]) Middleware(next http.Handler) http.Handler {
	mux := http.NewServeMux()

	pattern := path.Join(d.prefix, "{_REST_...}")

	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		rest := r.PathValue("_REST_")

		r.URL.Path = path.Join("/", rest)

		value, err := d.parser(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), dynamicPrefixKey{}, value)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
	return mux
}

func (d *DynamicPrefix[T]) Value(r *http.Request) T {
	value := r.Context().Value(dynamicPrefixKey{})

	if value == nil {
		var zero T
		return zero
	}

	return value.(T)
}
