package server

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Route struct {
	Path    string
	Method  string
	Handler http.Handler
}

func Routes(serv *httpServer) []Route {
	return []Route{
		{"/", "POST", http.HandlerFunc(serv.handleProduce)},
		{"/", "GET", http.HandlerFunc(serv.handleConsume)},
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("%v %v from %v", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(rw, r)
	})
}

func NewHttpServer(addr string) *http.Server {
	httpServer := newHttpServer()
	r := mux.NewRouter()

	for _, route := range Routes(httpServer) {
		r.Handle(route.Path, route.Handler).Methods(route.Method)
	}

	r.Use(LoggingMiddleware)

	return &http.Server{Addr: addr, Handler: r}
}
