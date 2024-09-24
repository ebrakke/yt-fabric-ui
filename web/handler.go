package web

import (
	"fmt"
	"html/template"
	"log"
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
}

func NewHandler(p *core.Processor, dataDir string) *Handler {
	h := &Handler{
		processor: p,
		dataDir:   dataDir,
	}
	h.setupRoutes()
	log.Println("Handler initialized")
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
	videos, err := core.LoadVideos(h.dataDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load videos: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/videos.html"))
	tmpl.Execute(w, map[string]interface{}{"Title": "Videos", "Videos": videos})
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/index.html"))
	tmpl.Execute(w, map[string]string{"Title": "Home"})
}

func (h *Handler) handleSubmitVideos(w http.ResponseWriter, r *http.Request) {
	videoLinks := r.FormValue("video_links")
	videoLinksList := strings.Split(videoLinks, "\n")
	for _, videoLink := range videoLinksList {
		h.processor.FetchVideo(videoLink)
	}
	fmt.Fprintf(w, "Videos processed: %d", len(videoLinksList))
}

func (h *Handler) handleVideoByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	video, err := core.LoadVideo(videoID, h.dataDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load video: %v", err), http.StatusInternalServerError)
		return
	}
	files, err := core.LoadVideoFiles(videoID, h.dataDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load video files: %v", err), http.StatusInternalServerError)
		return
	}
	models, err := core.ListModels()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load models: %v", err), http.StatusInternalServerError)
		return
	}
	patterns, err := core.ListPatterns()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load patterns: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/video.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]interface{}{"Title": "Video", "VideoID": videoID, "VideoTitle": video.Title, "Files": files, "Models": models, "Patterns": patterns})
}

func (h *Handler) handleVideoByIDSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]
	summary := vars["summary"]

	summary, err := core.LoadVideoSummary(videoID, h.dataDir, summary)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load video summary: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("layout.html").Funcs(templateFuncs).ParseFiles("web/templates/layout.html", "web/templates/video-summary.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]string{"Title": "Video", "VideoID": videoID, "Summary": summary})
}

func (h *Handler) handleProcessVideo(w http.ResponseWriter, r *http.Request) {
	videoID := r.FormValue("videoID")
	model := r.FormValue("model")
	pattern := r.FormValue("pattern")

	_, _, err := h.processor.ProcessVideo(videoID, model, pattern)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process video: %v", err), http.StatusInternalServerError)
		return
	}
	videoLink := fmt.Sprintf("/videos/%s/%s-%s.md", videoID, pattern, model)
	fmt.Fprintf(w, `<li><a href="%s" class="text-indigo-400 hover:text-indigo-300 transition duration-150 ease-in-out">%s-%s.md</a></li>`, videoLink, pattern, model)
}
