package middleware

import (
	"context"
	"net/http"
)

type pathParserKey struct{}

type PathParser[T any] struct {
	parser func(r *http.Request) (T, error)
}

func NewPathParser[T any](parser func(r *http.Request) (T, error)) *PathParser[T] {
	return &PathParser[T]{parser}
}

func (d *PathParser[T]) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, err := d.parser(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), pathParserKey{}, value)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (d *PathParser[T]) Value(r *http.Request) T {
	value := r.Context().Value(pathParserKey{})

	if value == nil {
		var zero T
		return zero
	}

	return value.(T)
}
