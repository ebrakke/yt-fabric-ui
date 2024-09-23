package main

import (
	"flag"
	"log"
	"net/http"

	"fabric-agents/core"
	"fabric-agents/web"
)

func main() {
	port := flag.String("port", "8080", "Port for the web server")
	flag.Parse()

	runWebServer(*port)
}

func runWebServer(port string) {
	handler := web.NewHandler(core.NewProcessor("data"), "data")
	http.Handle("/", handler)
	log.Printf("Starting web server on port %s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
