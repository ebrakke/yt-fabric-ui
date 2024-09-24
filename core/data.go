package core

import (
	"encoding/json"
	"fabric-agents/yt"
	"fmt"
	"os"
	"path/filepath"
)

// LoadVideos loads all videos from the data directory
func LoadVideos(dataDir string) ([]yt.Video, error) {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	var videos []yt.Video
	for _, file := range files {
		if file.IsDir() {
			video, err := LoadVideo(file.Name(), dataDir)
			if err != nil {
				return nil, err
			}
			videos = append(videos, video)
		}
	}
	return videos, nil
}

func SaveVideo(video yt.Video, dataDir string) error {
	videoDir := filepath.Join(dataDir, video.ID)
	os.MkdirAll(videoDir, 0755)
	transcriptPath := filepath.Join(videoDir, "data.json")
	videoJSON, err := json.Marshal(video)
	if err != nil {
		return err
	}

	return os.WriteFile(transcriptPath, videoJSON, 0644)
}

func SaveVideoFabricOutput(videoID string, output string, summary string, model string, dataDir string) error {
	videoDir := filepath.Join(dataDir, videoID)
	os.MkdirAll(videoDir, 0755)
	outputPath := filepath.Join(videoDir, fmt.Sprintf("%s-%s.md", summary, model))
	return os.WriteFile(outputPath, []byte(output), 0644)
}

func LoadVideo(videoID string, dataDir string) (yt.Video, error) {
	videoDir := filepath.Join(dataDir, videoID)
	transcriptPath := filepath.Join(videoDir, "data.json")
	videoJSON, err := os.ReadFile(transcriptPath)
	if err != nil {
		return yt.Video{}, err
	}

	var video yt.Video
	err = json.Unmarshal(videoJSON, &video)
	if err != nil {
		return yt.Video{}, err
	}

	return video, nil
}

func LoadVideoSummary(videoID string, dataDir string, summaryFileName string) (string, error) {
	videoDir := filepath.Join(dataDir, videoID)
	summaryPath := filepath.Join(videoDir, summaryFileName)
	summary, err := os.ReadFile(summaryPath)
	if err != nil {
		return "", err
	}
	return string(summary), nil
}

func LoadVideoFiles(videoID string, dataDir string) ([]string, error) {
	videoDir := filepath.Join(dataDir, videoID)
	files, err := os.ReadDir(videoDir)
	if err != nil {
		return nil, err
	}

	var filePaths []string
	for _, file := range files {
		if !file.IsDir() {
			filePaths = append(filePaths, file.Name())
		}
	}
	return filePaths, nil
}
