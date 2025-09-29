package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Middleware function type
type Middleware func(http.Handler) http.Handler

// HTTPError represents an error with an HTTP status code
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{Code: code, Message: message}
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

type Router struct {
	id          string
	mux         *http.ServeMux
	handler     http.Handler
	middlewares []Middleware
}

func NewRouter() *Router {
	mux := http.NewServeMux()
	return &Router{
		mux:         mux,
		handler:     mux,
		middlewares: []Middleware{},
		id:          "unknown",
	}
}

func (r *Router) ID(id string) {
	r.id = id
}

func (r *Router) Use(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

type RouterHandler func(r *http.Request) (any, error)
type Setup func(r *Router)

func (r *Router) GET(routePath string, handler RouterHandler, middlewares ...Middleware) {
	r.addRoute(http.MethodGet, routePath, handler, middlewares...)
}

func (r *Router) PATCH(routePath string, handler RouterHandler, middlewares ...Middleware) {
	r.addRoute(http.MethodPatch, routePath, handler, middlewares...)
}

func (r *Router) POST(routePath string, handler RouterHandler, middlewares ...Middleware) {
	r.addRoute(http.MethodPost, routePath, handler, middlewares...)
}

func (r *Router) PUT(routePath string, handler RouterHandler, middlewares ...Middleware) {
	r.addRoute(http.MethodPut, routePath, handler, middlewares...)
}

func (r *Router) DELETE(routePath string, handler RouterHandler, middlewares ...Middleware) {
	r.addRoute(http.MethodDelete, routePath, handler, middlewares...)
}

func (r *Router) Mount(prefix string, register func(*Router), middlewares ...Middleware) {
	prefix = strings.TrimSuffix(prefix, "/")
	child := NewRouter()

	register(child)

	r.mux.Handle(prefix+"/", http.StripPrefix(prefix, child))
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	modReq := *req
	modReq.URL = new(url.URL)
	*modReq.URL = *req.URL

	if !strings.HasSuffix(modReq.URL.Path, "/") {
		modReq.URL.Path += "/"
	}

	var handler http.Handler
	handler = r.mux

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	handler.ServeHTTP(w, &modReq)
}

func (r *Router) addRoute(method, routePath string, handler RouterHandler, middlewares ...Middleware) {
	if routePath == "/" {
		routePath = path.Join(routePath, "{$}")
	}

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := handler(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			if httpErr, ok := err.(*HTTPError); ok {
				w.WriteHeader(httpErr.Code)
				json.NewEncoder(w).Encode(httpErr)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "Failed to encode response"})
		}
	})

	r.mux.Handle(method+" "+routePath, wrappedHandler)
}

// // Normalize trailing slashes
// if strings.TrimPrefix(modReq.URL.Path, prefix+"/") != "" {
// 	if modReq.URL.Path == modReq.RequestURI {
// 		modReq.URL.Path = modReq.URL.Path + "/"
// 	} else {
// 		modReq.URL.Path = strings.TrimSuffix(modReq.URL.Path, "/")
// 	}
// }
