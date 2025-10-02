package router

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Middleware func(http.Handler) http.Handler
type RouterHandler func(w *ResponseWriter, r *http.Request) error
type Setup func(r *Router)

type Router struct {
	mux         *http.ServeMux
	middlewares []*Middleware
}

func NewRouter() *Router {
	mux := http.NewServeMux()
	return &Router{
		mux:         mux,
		middlewares: []*Middleware{},
	}
}

func (r *Router) Use(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, pointers(middlewares)...)
}

func (r *Router) Mount(prefix string, register func(*Router), middlewares ...Middleware) {
	prefix = strings.TrimSuffix(prefix, "/")
	child := NewRouter()
	child.middlewares = r.middlewares

	register(child)

	r.mux.Handle(prefix+"/", http.StripPrefix(prefix, child))
}

func (r *Router) WithPrefix(prefix string, options ...Middleware) *Router {
	prefix = strings.TrimSuffix(prefix, "/")
	child := NewRouter()
	child.middlewares = r.middlewares

	r.mux.Handle(prefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pattern := strings.Split(strings.TrimSuffix(r.Pattern, "/"), "/")
		prefix := strings.Join(strings.Split(r.URL.Path, "/")[:len(pattern)], "/")

		handler := applyMiddleware(child, pointers(options))
		http.StripPrefix(prefix, handler).ServeHTTP(w, r)
	}))

	return child
}

func (r *Router) Do(do func(r *Router)) {
	do(r)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	modReq := *req
	modReq.URL = new(url.URL)
	*modReq.URL = *req.URL

	if !strings.HasSuffix(modReq.URL.Path, "/") {
		modReq.URL.Path += "/"
	}

	muxHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, pattern := r.mux.Handler(req)

		// if no route matched, apply middleware to handle 404
		if pattern == "" {
			applyMiddleware(r.mux, r.middlewares).ServeHTTP(w, req)
			return
		}

		r.mux.ServeHTTP(w, req)
	})

	muxHandler.ServeHTTP(w, &modReq)
}

func (r *Router) GET(routePath string, handler RouterHandler, options ...Middleware) {
	r.addRoute(http.MethodGet, routePath, handler, options...)
}

func (r *Router) PATCH(routePath string, handler RouterHandler, options ...Middleware) {
	r.addRoute(http.MethodPatch, routePath, handler, options...)
}

func (r *Router) POST(routePath string, handler RouterHandler, options ...Middleware) {
	r.addRoute(http.MethodPost, routePath, handler, options...)
}

func (r *Router) PUT(routePath string, handler RouterHandler, options ...Middleware) {
	r.addRoute(http.MethodPut, routePath, handler, options...)
}

func (r *Router) DELETE(routePath string, handler RouterHandler, options ...Middleware) {
	r.addRoute(http.MethodDelete, routePath, handler, options...)
}

func (r *Router) addRoute(method, routePath string, routeHandler RouterHandler, options ...Middleware) {
	// TODO - figure out the slash stuff once and for all
	if routePath == "/" {
		routePath = path.Join(routePath, "{$}")
	}

	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeHandler(&ResponseWriter{ResponseWriter: w}, r)
	})

	handler := applyMiddleware(handlerFunc, append(pointers(options), r.middlewares...))

	r.mux.Handle(method+" "+routePath, handler)
}

func applyMiddleware[T ~func(http.Handler) http.Handler](handler http.Handler, middlewares []*T) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := *middlewares[i]
		handler = middleware(handler)
	}

	return handler
}

func pointers[T any](values []T) []*T {
	pointers := make([]*T, len(values))
	for i, value := range values {
		pointers[i] = &value
	}
	return pointers
}
