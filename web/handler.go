package web

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"fabric-agents/core"

	"github.com/gorilla/mux"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var templateFuncs = template.FuncMap{
	"markdown": func(text string) template.HTML {
		html := blackfriday.Run([]byte(text))
		return template.HTML(html)
	},
	"formatVideoTitle": func(title string) template.HTML {
		formattedTitle := cases.Title(language.English, cases.Compact).String(strings.ReplaceAll(title, "-", " "))
		return template.HTML(fmt.Sprintf("<span class='text-2xl font-bold text-indigo-400'>%s</span>", formattedTitle))
	},
}

type Handler struct {
	processor *core.Processor
	router    *mux.Router
	dataDir   string
	logger    *slog.Logger
}

func NewHandler(p *core.Processor, dataDir string, logger *slog.Logger) *Handler {
	h := &Handler{
		processor: p,
		dataDir:   dataDir,
		logger:    logger,
	}
	h.setupRoutes()
	h.logger.Info("Handler initialized")
	return h
}

func (h *Handler) setupRoutes() {
	h.router = mux.NewRouter()
	h.router.HandleFunc("/", h.handleIndex)
	h.router.HandleFunc("/submit-videos", h.handleSubmitVideos)
	h.router.HandleFunc("/videos", h.handleVideos)
	h.router.HandleFunc("/process-video", h.handleProcessVideo)
	h.router.HandleFunc("/videos/{id}", h.handleVideoByID)
	h.router.HandleFunc("/videos/{id}/{summary}", h.handleVideoByIDSummary)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleVideos(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Handling /videos request")
	videos, err := core.LoadVideos(h.dataDir)
	if err != nil {
		h.logger.Error("Failed to load videos", "error", err)
		http.Error(w, fmt.Sprintf("Failed to load videos: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/videos.html"))
	tmpl.Execute(w, map[string]interface{}{"Title": "Videos", "Videos": videos})
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Handling / request")
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/index.html"))
	tmpl.Execute(w, map[string]string{"Title": "Home"})
}

func (h *Handler) handleSubmitVideos(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Handling /submit-videos request")
	videoLinks := r.FormValue("video_links")
	videoLinksList := strings.Split(videoLinks, "\n")
	for _, videoLink := range videoLinksList {
		h.logger.Info("Processing video link", "link", videoLink)
		h.processor.FetchVideo(videoLink)
	}
	h.logger.Info("Videos processed", "count", len(videoLinksList))
	fmt.Fprintf(w, "Videos processed: %d", len(videoLinksList))
}

func (h *Handler) handleVideoByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]
	h.logger.Debug("Handling /videos/{id} request", "videoID", videoID)

	// Check if this is a delete request
	if r.Method == "DELETE" {
		h.logger.Info("Deleting video", "videoID", videoID)
		err := core.DeleteVideo(videoID, h.dataDir)
		if err != nil {
			h.logger.Error("Failed to delete video", "videoID", videoID, "error", err)
			http.Error(w, fmt.Sprintf("Failed to delete video: %v", err), http.StatusInternalServerError)
			return
		}
		// Redirect to the videos list page after successful deletion
		w.Header().Set("HX-Redirect", "/videos")
		return
	}

	video, err := core.LoadVideo(videoID, h.dataDir)
	if err != nil {
		h.logger.Error("Failed to load video", "videoID", videoID, "error", err)
		http.Error(w, fmt.Sprintf("Failed to load video: %v", err), http.StatusInternalServerError)
		return
	}
	files, err := core.LoadVideoFiles(videoID, h.dataDir)
	if err != nil {
		h.logger.Error("Failed to load video files", "videoID", videoID, "error", err)
		http.Error(w, fmt.Sprintf("Failed to load video files: %v", err), http.StatusInternalServerError)
		return
	}
	models, err := core.ListModels()
	if err != nil {
		h.logger.Error("Failed to load models", "error", err)
		http.Error(w, fmt.Sprintf("Failed to load models: %v", err), http.StatusInternalServerError)
		return
	}
	patterns, err := core.ListPatterns()
	if err != nil {
		h.logger.Error("Failed to load patterns", "error", err)
		http.Error(w, fmt.Sprintf("Failed to load patterns: %v", err), http.StatusInternalServerError)
		return
	}
	savedPatterns, err := core.LoadPatterns()
	if err != nil {
		h.logger.Error("Failed to load saved patterns", "error", err)
		http.Error(w, fmt.Sprintf("Failed to load patterns: %v", err), http.StatusInternalServerError)
		return
	}
	savedModels, err := core.LoadModels()
	if err != nil {
		h.logger.Error("Failed to load saved models", "error", err)
		http.Error(w, fmt.Sprintf("Failed to load models: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/video.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, map[string]interface{}{
		"Title":       "Video",
		"VideoID":     videoID,
		"VideoTitle":  video.Title,
		"Files":       files,
		"Models":      savedModels,
		"Patterns":    savedPatterns,
		"AllModels":   models,
		"AllPatterns": patterns,
	})
	if err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		http.Error(w, fmt.Sprintf("Failed to execute template: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) handleVideoByIDSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]
	summary := vars["summary"]
	h.logger.Debug("Handling /videos/{id}/{summary} request", "videoID", videoID, "summary", summary)

	summary, err := core.LoadVideoSummary(videoID, h.dataDir, summary)
	if err != nil {
		h.logger.Error("Failed to load video summary", "videoID", videoID, "summary", summary, "error", err)
		http.Error(w, fmt.Sprintf("Failed to load video summary: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("layout.html").Funcs(templateFuncs).ParseFiles("web/templates/layout.html", "web/templates/video-summary.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]string{"Title": "Video", "VideoID": videoID, "Summary": summary})
}

func (h *Handler) handleProcessVideo(w http.ResponseWriter, r *http.Request) {
	videoID := r.FormValue("videoID")
	model := r.FormValue("model")
	pattern := r.FormValue("pattern")
	h.logger.Debug("Handling /process-video request", "videoID", videoID, "model", model, "pattern", pattern)

	_, _, err := h.processor.ProcessVideo(videoID, model, pattern)
	if err != nil {
		h.logger.Error("Failed to process video", "videoID", videoID, "model", model, "pattern", pattern, "error", err)
		http.Error(w, fmt.Sprintf("Failed to process video: %v", err), http.StatusInternalServerError)
		return
	}
	videoLink := fmt.Sprintf("/videos/%s/%s-%s.md", videoID, pattern, model)
	fmt.Fprintf(w, `<li><a href="%s" class="text-indigo-400 hover:text-indigo-300 transition duration-150 ease-in-out">%s-%s.md</a></li>`, videoLink, pattern, model)
}
