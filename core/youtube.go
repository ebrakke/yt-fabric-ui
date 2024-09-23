package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Video struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Transcript string `json:"transcript"`
}

// GetTranscriptFromURL gets the transcript from a YouTube video URL
func GetTranscriptFromURL(videoURL string) (Video, error) {
	// Execute the yt command to get the transcript
	cmd := exec.Command("yt", "--transcript", videoURL)
	output, err := cmd.Output()
	if err != nil {
		return Video{}, fmt.Errorf("failed to get transcript: %v", err)
	}
	// Check if the output contains "transcript not found"
	if strings.Contains(strings.ToLower(string(output)), "transcript not found") {
		return Video{}, fmt.Errorf("no transcript available for this video: %s", videoURL)
	}

	videoID, err := GetVideoIDFromURL(videoURL)
	if err != nil {
		return Video{}, fmt.Errorf("failed to get video ID: %v", err)
	}

	title, err := GetVideoTitleFromID(videoID)
	if err != nil {
		return Video{}, fmt.Errorf("failed to get video title: %v", err)
	}

	return Video{
		ID:         videoID,
		Title:      title,
		Transcript: string(output),
	}, nil

}

// GetVideoIDFromURL gets the video ID from a YouTube video URL
func GetVideoIDFromURL(videoURL string) (string, error) {
	url, err := url.Parse(videoURL)
	if err != nil {
		return "", err
	}

	videoID := url.Query().Get("v")
	if videoID == "" {
		return "", fmt.Errorf("no video ID found in URL: %s", videoURL)
	}
	return videoID, nil
}

// GetVideoTitleFromID gets the video title from a YouTube video ID
func GetVideoTitleFromID(videoID string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch YouTube page: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
