package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Processor struct {
	transcriptsDir string
}

func NewProcessor(transcriptsDir string) *Processor {
	return &Processor{transcriptsDir: transcriptsDir}
}

func (p *Processor) RunPattern(pattern, outputFile, model string, limit int) error {
	files, err := os.ReadDir(p.transcriptsDir)
	if err != nil {
		return err
	}

	if limit > 0 && limit < len(files) {
		files = files[:limit]
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		transcriptPath := filepath.Join(p.transcriptsDir, file.Name())
		transcript, err := os.ReadFile(transcriptPath)
		if err != nil {
			return fmt.Errorf("error reading file %s: %v", transcriptPath, err)
		}

		title := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		title = strings.SplitN(title, "_", 2)[1]

		if err := os.MkdirAll(title, os.ModePerm); err != nil {
			return fmt.Errorf("error creating output directory for %s: %v", title, err)
		}

		var cmd *exec.Cmd
		if model != "" {
			cmd = exec.Command("fabric", "--pattern", pattern, "--model", model)
		} else {
			cmd = exec.Command("fabric", "--pattern", pattern)
		}
		cmd.Stdin = strings.NewReader(string(transcript))

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error executing fabric pattern for %s: %v", title, err)
		}

		outputPath := filepath.Join(title, outputFile+".md")
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return fmt.Errorf("error writing output for %s: %v", title, err)
		}

		fmt.Printf("Processed %s and saved ideas to %s\n", file.Name(), outputPath)
	}

	return nil
}

func (p *Processor) FetchTranscript(videoLink string) (string, error) {
	// Execute the yt command to get the transcript
	cmd := exec.Command("yt", "--transcript", videoLink)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get transcript: %v", err)
	}
	// Check if the output contains "transcript not found"
	if strings.Contains(strings.ToLower(string(output)), "transcript not found") {
		return "", fmt.Errorf("no transcript available for this video: %s", videoLink)
	}

	// Return the transcript as a string
	return string(output), nil
}

func (p *Processor) SaveTranscript(videoLink, transcript string) error {
	parsed, err := url.Parse(videoLink)
	if err != nil {
		return fmt.Errorf("failed to parse video link: %v", err)
	}
	videoID := parsed.Query().Get("v")

	// Get the video title (you need to implement this function)
	videoTitle, err := getYouTubeVideoTitle(videoID)
	if err != nil {
		return fmt.Errorf("failed to get video title: %v", err)
	}

	// Sanitize the title to make it safe for filenames
	safeTitle := sanitizeFilename(videoTitle)

	filename := fmt.Sprintf("%s_%s.txt", videoID, safeTitle)
	path := filepath.Join(p.transcriptsDir, filename)

	err = os.MkdirAll(p.transcriptsDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create transcriptions directory: %v", err)
	}

	return os.WriteFile(path, []byte(transcript), 0644)
}

// Helper function to sanitize the filename
func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalid {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	// Trim spaces from the beginning and end
	return strings.TrimSpace(filename)
}

func (p *Processor) GetTranscripts() ([]string, error) {
	files, err := os.ReadDir(p.transcriptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcripts directory: %v", err)
	}

	var transcripts []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			// Strip the .txt extension from the filename
			baseName := strings.TrimSuffix(file.Name(), ".txt")
			transcripts = append(transcripts, baseName)
		}
	}

	return transcripts, nil
}

func (p *Processor) GetTranscript(fileName string) (string, error) {
	files, err := os.ReadDir(p.transcriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read transcripts directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), fileName) {
			transcript, err := os.ReadFile(filepath.Join(p.transcriptsDir, file.Name()))
			if err != nil {
				return "", fmt.Errorf("failed to read transcript file: %v", err)
			}
			return string(transcript), nil
		}
	}

	return "", fmt.Errorf("transcript not found for video ID: %s", fileName)
}

func (p *Processor) ProcessTranscript(videoLink string) (string, error) {
	transcript, err := p.GetTranscript(videoLink)
	if err != nil {
		return "", fmt.Errorf("failed to fetch transcript: %v", err)
	}

	// Prepare the fabric CLI command
	cmd := exec.Command("fabric", "--pattern", "summarize", transcript)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fabric CLI error: %v", err)
	}

	// Create the output directory if it doesn't exist
	outputDir := filepath.Join("processed", videoLink)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Write the processed output to summary.md
	outputPath := filepath.Join(outputDir, "summary.md")
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return "", fmt.Errorf("failed to write summary file: %v", err)
	}

	// Return the processed transcript
	return string(output), nil
}

type ProcessedVideo struct {
	Title     string
	Summaries []string
	ID        string
}

func (p *Processor) GetProcessedVideos() ([]ProcessedVideo, error) {
	files, err := os.ReadDir("processed")
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %v", err)
	}

	var processedVideos []ProcessedVideo
	for _, file := range files {
		if file.IsDir() {
			summaries, err := p.getSummariesForVideo(file.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to get summaries for video %s: %v", file.Name(), err)
			}
			processedVideos = append(processedVideos, ProcessedVideo{
				Title:     file.Name(),
				Summaries: summaries,
				ID:        file.Name(),
			})
		}
	}
	return processedVideos, nil
}

func (p *Processor) getSummariesForVideo(videoDir string) ([]string, error) {
	summaryDir := filepath.Join("processed", videoDir)
	files, err := os.ReadDir(summaryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read summary directory for video %s: %v", videoDir, err)
	}

	var summaries []string
	for _, file := range files {
		if !file.IsDir() {
			summaries = append(summaries, file.Name())
		}
	}
	return summaries, nil
}

func getYouTubeVideoTitle(videoID string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch YouTube page: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
	matches := titleRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("title not found in YouTube page")
	}

	title := string(matches[1])
	title = strings.TrimSuffix(title, " - YouTube")
	title = strings.ToLower(title)
	title = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, title)
	title = strings.Trim(title, "-")
	title = regexp.MustCompile(`-+`).ReplaceAllString(title, "-")

	return title, nil
}
