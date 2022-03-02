package main

import (
	"log"

	"github.com/arindas/proglog/internal/server"
)

func main() {
	server := server.NewHttpServer(":8080")
	log.Fatal(server.ListenAndServe())
}
