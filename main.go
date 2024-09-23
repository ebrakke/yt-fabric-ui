package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"fabric-agents/cli"
	"fabric-agents/core"
	"fabric-agents/web"
)

func main() {
	// Define flags
	transcribeMode := flag.Bool("transcribe", false, "Run in transcribe mode to transcribe YouTube links")
	patternMode := flag.String("pattern", "", "Run in pattern mode to run a fabric pattern")
	searchMode := flag.String("search", "", "Run in search mode to find YouTube videos")
	modelFlag := flag.String("model", "", "Specify the model to use with fabric")
	limit := flag.Int("limit", 10, "Limit the number of videos to process")
	webMode := flag.Bool("web", false, "Run in web server mode")
	port := flag.String("port", "8080", "Port for the web server")
	flag.Parse()

	if *webMode {
		runWebServer(*port)
	} else {
		runCLI(*transcribeMode, *patternMode, *searchMode, *modelFlag, *limit)
	}
}

func runWebServer(port string) {
	handler := web.NewHandler(core.NewProcessor("transcripts"))
	http.Handle("/", handler)
	log.Printf("Starting web server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runCLI(transcribeMode bool, patternMode, searchMode, modelFlag string, limit int) {
	processor := core.NewProcessor("transcripts")
	if transcribeMode {
		cli.RunTranscribeMode(processor)
	} else if patternMode != "" {
		cli.RunPatternMode(processor, patternMode, modelFlag, limit)
	} else if searchMode != "" {
		cli.RunSearchMode(processor, searchMode)
	} else {
		fmt.Println("No mode specified. Use --transcribe, --pattern, --search, or --web to run in a specific mode.")
		flag.PrintDefaults()
	}
}
