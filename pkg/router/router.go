package router

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Middleware func(http.Handler) http.Handler

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

type RouterHandler func(w *ResponseWriter, r *http.Request) error

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

	applyMiddleware(r.mux, r.middlewares).ServeHTTP(w, &modReq)
}

func (r *Router) addRoute(method, routePath string, handler RouterHandler, middlewares ...Middleware) {
	if routePath == "/" {
		routePath = path.Join(routePath, "{$}")
	}

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(&ResponseWriter{ResponseWriter: w}, r)
	})

	r.mux.Handle(method+" "+routePath, applyMiddleware(wrappedHandler, middlewares))
}

func applyMiddleware(handler http.Handler, middlewares []Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

// // Normalize trailing slashes
// if strings.TrimPrefix(modReq.URL.Path, prefix+"/") != "" {
// 	if modReq.URL.Path == modReq.RequestURI {
// 		modReq.URL.Path = modReq.URL.Path + "/"
// 	} else {
// 		modReq.URL.Path = strings.TrimSuffix(modReq.URL.Path, "/")
// 	}
// }
