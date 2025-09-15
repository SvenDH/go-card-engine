package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()

	// Serve static files from the public directory
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	log.Printf("Server starting on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}