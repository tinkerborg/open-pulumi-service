package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tinkerborg/open-pulumi-service/pkg/router"
)

type claimsKey struct{}
type tokenTypeKey struct{}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.Split(r.Header.Get("Authorization"), " ")

		// tokenType := header[0]
		token := header[1]

		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := s.GetUserClaims(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler := s.getTokenTypeHandler(r)

		if handler != nil {
			if handler.tokenType != claims.Type {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if !handler.checkClaims(r, claims) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		ctx := context.WithValue(r.Context(), claimsKey{}, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type CheckClaims func(r *http.Request, claims *UserClaims) bool

type tokenTypeHandler struct {
	tokenType   string
	checkClaims CheckClaims
}

func (s *Service) WithTokenType(tokenType string, checkClaims CheckClaims) router.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler := &tokenTypeHandler{tokenType, checkClaims}
			ctx := context.WithValue(r.Context(), tokenTypeKey{}, handler)
			fmt.Printf("SET TOKEN TYPE %s\n", tokenType)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *Service) GetRequestClaims(r *http.Request) (*UserClaims, error) {
	value := r.Context().Value(claimsKey{})

	if value == nil {
		return nil, errors.New("request missing user claim")
	}

	return value.(*UserClaims), nil
}

func (s *Service) getTokenTypeHandler(r *http.Request) *tokenTypeHandler {
	value := r.Context().Value(tokenTypeKey{})
	if value == nil {
		return nil
	}

	return value.(*tokenTypeHandler)
}
