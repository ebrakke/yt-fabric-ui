package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"fabric-agents/core"
	"fabric-agents/web"
	"fabric-agents/yt"
	"log/slog"
)

func main() {
	// Initialize the logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Set the default logging level
	})
	logger := slog.New(logHandler)

	port := flag.String("port", "8080", "Port for the web server")
	flag.Parse()

	runWebServer(*port, logger)
}

func runWebServer(port string, logger *slog.Logger) {
	processor := core.NewProcessor(logger, "data/videos", yt.NewYT(""))
	handler := web.NewHandler(processor, "data/videos", logger)
	http.Handle("/", handler)
	logger.Info("Starting web server", "port", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
