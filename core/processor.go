package core

import (
	"fmt"
	"os"
	"path/filepath"
)

type Processor struct {
	filesDir string
}

func NewProcessor(filesDir string) *Processor {
	return &Processor{filesDir: filesDir}
}

// ProcessVideo processes a video and returns the video directory
func (p *Processor) FetchVideoTranscript(videoLink string) (string, error) {
	videoID, err := GetVideoIDFromURL(videoLink)
	fmt.Println("Video ID:", videoID)
	if err != nil {
		return "", fmt.Errorf("failed to get video ID: %v", err)
	}
	os.MkdirAll(filepath.Join(p.filesDir, videoID), 0755)

	dataPath := filepath.Join(p.filesDir, videoID, "data.json")
	if _, err := os.Stat(dataPath); err == nil {
		// Transcript already exists, return the existing videoDir
		return videoID, nil
	}

	video, err := GetTranscriptFromURL(videoLink)
	if err != nil {
		return "", fmt.Errorf("failed to get transcript: %v", err)
	}

	SaveVideo(video, p.filesDir)
	return video.ID, nil
}

func (p *Processor) ProcessVideo(videoID string, model string, pattern string) (string, error) {
	video, err := LoadVideo(videoID, p.filesDir)
	if err != nil {
		return "", fmt.Errorf("failed to load video: %v", err)
	}

	output, err := RunFabric(video.Transcript, pattern, model)
	if err != nil {
		fmt.Println("Failed to run fabric:", err)
		return "", fmt.Errorf("failed to run fabric: %v", err)
	}

	SaveVideoFabricOutput(videoID, output, pattern, model, p.filesDir)
	return output, nil
}
