package cli

import (
	"fmt"
	"log"
	"os"

	"fabric-agents/core"
)

func RunTranscribeMode(p *core.Processor) {
	if len(os.Args) < 3 {
		fmt.Println("Please provide a file to parse.")
		return
	}
	fileToParse := os.Args[2]
	// Implement transcribe mode using the processor
	fmt.Printf("Running transcribe mode with file: %s\n", fileToParse)
}

func RunPatternMode(p *core.Processor, pattern, model string, limit int) {
	err := p.RunPattern(pattern, pattern, model, limit)
	if err != nil {
		log.Fatalf("Error running pattern mode: %v", err)
	}
}

func RunSearchMode(p *core.Processor, searchTerm string) {
	// Implement search mode using the processor
	fmt.Printf("Running search mode for term: %s\n", searchTerm)
}
