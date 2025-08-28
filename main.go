package main

import (
	"log"
	"net/http"
	"time"

	"github.com/frhan23/chat-go/routes"
	"github.com/gorilla/mux"
)

func main() {
	route := mux.NewRouter()
	routes.NewRoutes(route)

	listener := &http.Server{
		Addr:         ":5000",
		Handler:      route,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	defer func() {
		if r := recover(); r != nil {
			log.Print("recovered: error:", r)
		}
		listener.Close()
	}()
	log.Print("listening at " + listener.Addr)
	log.Fatal(listener.ListenAndServe())
}
