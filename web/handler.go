package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fabric-agents/core"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/gorilla/mux"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Handler struct {
	processor  *core.Processor
	videoQueue chan string
	queueWG    sync.WaitGroup
	router     *mux.Router
}

func NewHandler(p *core.Processor) *Handler {
	h := &Handler{
		processor:  p,
		videoQueue: make(chan string, 100), // Buffer size of 100, adjust as needed
	}
	h.setupRoutes()
	go h.processVideoQueue()
	log.Println("Handler initialized")
	return h
}

func (h *Handler) setupRoutes() {
	h.router = mux.NewRouter()
	h.router.HandleFunc("/", h.handleIndex)
	h.router.HandleFunc("/transcripts", h.handleTranscripts)
	h.router.HandleFunc("/transcripts/{id}", h.handleTranscript)
	h.router.HandleFunc("/submit-videos", h.handleSubmitVideos)
	h.router.HandleFunc("/process/{id}", h.handleProcessTranscript)
	h.router.HandleFunc("/processed/{id}", h.handleProcessedTranscript)
	h.router.HandleFunc("/processed/{id}/{summary}", h.handleProcessedSummary)
	h.router.HandleFunc("/processed", h.handleProcessedList)
	h.router.HandleFunc("/transcript/", transcriptHandler)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/index.html"))
	tmpl.Execute(w, map[string]string{"Title": "Home"})
}

func (h *Handler) handleProcessTranscript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptID := vars["id"]
	log.Printf("Processing transcript with ID: %s", transcriptID)

	// Process the transcript
	_, err := h.processor.ProcessTranscript(transcriptID)
	if err != nil {
		log.Printf("Failed to process transcript %s: %v", transcriptID, err)
		http.Error(w, fmt.Sprintf("Failed to process transcript: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Transcript %s processed successfully", transcriptID)
	http.Redirect(w, r, fmt.Sprintf("/processed/%s", transcriptID), http.StatusSeeOther)
}

func (h *Handler) handleProcessedSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptID := vars["id"]
	summaryID := vars["summary"]
	log.Printf("Rendering processed summary with ID: %s", summaryID)

	outputDir := filepath.Join("processed", transcriptID)
	outputPath := filepath.Join(outputDir, summaryID)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read processed summary: %v", err), http.StatusInternalServerError)
		return
	}

	// Render the processed summary
	funcMap := template.FuncMap{
		"markdown": renderMarkdown,
	}

	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles("web/templates/layout.html", "web/templates/processed.html")
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}
	videoTitle := extractVideoTitle(transcriptID)
	tmpl.Execute(w, map[string]interface{}{
		"Title":        "Processed Summary",
		"TranscriptID": transcriptID,
		"VideoURL":     fmt.Sprintf("https://www.youtube.com/watch?v=%s", transcriptID),
		"VideoTitle":   videoTitle,
		"Content":      string(content),
	})
}

func (h *Handler) handleProcessedTranscript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptID := vars["id"]
	log.Printf("Rendering processed transcript with ID: %s", transcriptID)

	outputDir := filepath.Join("processed", transcriptID)
	files, err := os.ReadDir(outputDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read processed transcript: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/video.html"))
	videoID := getVideoIDFromTranscriptID(transcriptID)
	tmpl.Execute(w, map[string]interface{}{
		"Title":        "Processed Transcript",
		"TranscriptID": transcriptID,
		"VideoID":      videoID,
		"VideoURL":     fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		"VideoTitle":   extractVideoTitle(transcriptID),
		"Files":        files,
	})
}

func (h *Handler) handleTranscript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptName := vars["id"]
	transcript, err := h.processor.GetTranscript(transcriptName)
	if err != nil {
		http.Error(w, "Failed to get transcript", http.StatusInternalServerError)
		return
	}

	// Parse the template files
	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/transcript.html")
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, map[string]interface{}{
		"Title":        "Transcript",
		"Transcript":   transcript,
		"TranscriptID": transcriptName,
	})
}

func (h *Handler) handleTranscripts(w http.ResponseWriter, r *http.Request) {
	transcripts, err := h.processor.GetTranscripts()
	if err != nil {
		http.Error(w, "Failed to get transcripts", http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/transcripts.html"))
	tmpl.Execute(w, map[string]interface{}{"Title": "Transcripts", "Transcripts": transcripts})
}

func (h *Handler) handleSubmitVideos(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	videoLinks := r.FormValue("video_links")
	links := strings.Split(videoLinks, "\n")

	log.Printf("Received %d video links for processing", len(links))

	for _, link := range links {
		link = strings.TrimSpace(link)
		if link != "" {
			h.videoQueue <- link
			h.queueWG.Add(1)
			log.Printf("Added video link to queue: %s", link)
		}
	}

	response := fmt.Sprintf("Received %d video links for processing", len(links))
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(response))
}

func (h *Handler) processVideoQueue() {
	for link := range h.videoQueue {
		go func(link string) {
			defer h.queueWG.Done()

			log.Printf("Processing video link: %s", link)

			// Get the transcript
			transcript, err := h.processor.FetchTranscript(link)
			if err != nil {
				log.Printf("Error processing video %s: %v", link, err)
				return
			}

			// Save the transcript
			err = h.processor.SaveTranscript(link, transcript)
			if err != nil {
				log.Printf("Error saving transcript for video %s: %v", link, err)
			} else {
				log.Printf("Transcript saved successfully for video: %s", link)
			}
		}(link)
	}
}

func renderMarkdown(input string) template.HTML {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)
	md := []byte(input)
	output := markdown.ToHTML(md, parser, nil)
	return template.HTML(output)
}

func (h *Handler) handleProcessedList(w http.ResponseWriter, r *http.Request) {
	processedVideos, err := h.processor.GetProcessedVideos()
	if err != nil {
		http.Error(w, "Failed to get processed videos", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/processed-list.html"))
	tmpl.Execute(w, map[string]interface{}{"Title": "Processed Videos", "ProcessedVideos": processedVideos})
}

func extractVideoTitle(transcriptID string) string {
	videoTitle := strings.ReplaceAll(transcriptID, "_", " ")
	videoTitle = strings.ReplaceAll(videoTitle, "-", " ")
	videoTitle = cases.Title(language.English).String(videoTitle)
	return videoTitle
}

func transcriptHandler(w http.ResponseWriter, r *http.Request) {
	videoID := r.URL.Path[len("/transcript/"):]
	transcriptPath := filepath.Join("transcripts", videoID+".txt")

	content, err := os.ReadFile(transcriptPath)
	if err != nil {
		http.Error(w, "Transcript not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func getVideoIDFromTranscriptID(transcriptID string) string {
	videoID := strings.Split(transcriptID, "_")[0]
	return videoID
}
