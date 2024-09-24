package yt

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/anaskhan96/soup"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type YT struct {
	apiKey  string
	service *youtube.Service
}

type Video struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Channel    string   `json:"channel"`
	Transcript string   `json:"transcript"`
	Comments   []string `json:"comments"`
	Duration   int      `json:"duration"`
	URL        string   `json:"url"`
}

func NewYT(apiKey string) *YT {
	var service *youtube.Service
	var err error
	if apiKey == "" {
		service = nil
	} else {
		service, err = youtube.NewService(context.Background(), option.WithAPIKey(apiKey))
		if err != nil {
			log.Fatalf("Error creating YouTube client: %v", err)
		}
	}
	return &YT{apiKey: apiKey, service: service}
}

func (y *YT) GetVideoInfo(url string) (*Video, error) {
	fmt.Println("Getting video info for", url)
	videoID := y.GetVideoID(url)
	if videoID == "" {
		return nil, fmt.Errorf("invalid YouTube URL")
	}

	output := &Video{
		ID: videoID,
	}
	videoDetails, err := y.getVideoDetails(videoID)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	output.Transcript = videoDetails["transcript"]
	output.Title = videoDetails["title"]
	output.Channel = videoDetails["channel"]
	return output, nil
}

func (y *YT) GetVideoID(url string) string {
	pattern := `(?:https?:\/\/)?(?:www\.)?(?:youtube\.com\/(?:[^\/\n\s]+\/\S+\/|(?:v|e(?:mbed)?)\/|\S*?[?&]v=)|youtu\.be\/)([a-zA-Z0-9_-]{11})`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func (y *YT) getVideoDetails(videoID string) (map[string]string, error) {
	url := "https://www.youtube.com/watch?v=" + videoID
	resp, err := soup.Get(url)
	if err != nil {
		return nil, err
	}

	doc := soup.HTMLParse(resp)

	transcript, err := y.getTranscript(doc)
	if err != nil {
		return nil, err
	}
	title := getTitle(doc)
	channel := getCreator(doc)

	return map[string]string{
		"transcript": transcript,
		"title":      title,
		"channel":    channel,
	}, nil
}

func (y *YT) getTranscript(doc soup.Root) (string, error) {
	scriptTags := doc.FindAll("script")
	for _, scriptTag := range scriptTags {
		if strings.Contains(scriptTag.Text(), "captionTracks") {
			regex := regexp.MustCompile(`"captionTracks":(\[.*?\])`)
			match := regex.FindStringSubmatch(scriptTag.Text())
			if len(match) > 1 {
				var captionTracks []struct {
					BaseURL string `json:"baseUrl"`
				}
				json.Unmarshal([]byte(match[1]), &captionTracks)
				if len(captionTracks) > 0 {
					transcriptURL := captionTracks[0].BaseURL
					transcriptResp, err := soup.Get(transcriptURL)

					if err != nil {
						return "", err
					}
					transcript, err := unmarshalTranscript([]byte(transcriptResp))
					if err != nil {
						return "", err
					}
					var transcriptLines []string
					for _, track := range transcript.Texts {
						cleanedText := strings.ReplaceAll(track.Value, "&#39;", "'")
						transcriptLines = append(transcriptLines, cleanedText)
					}

					cleanedTranscript := strings.Join(transcriptLines, " ")
					cleanedTranscript = cleanedTranscript + "\n"
					return cleanedTranscript, nil
				}
			}
		}
	}
	return "", fmt.Errorf("transcript not found")
}

func getTitle(doc soup.Root) string {
	title := doc.Find("title").Text()
	title = strings.TrimSuffix(title, " - YouTube")
	return title
}

func getCreator(doc soup.Root) string {
	scriptTags := doc.FindAll("script")
	for _, scriptTag := range scriptTags {
		if strings.Contains(scriptTag.Text(), "ownerChannelName") {
			regex := regexp.MustCompile(`"ownerChannelName":"(.*?)"`)
			match := regex.FindStringSubmatch(scriptTag.Text())
			if len(match) > 1 {
				return match[1]
			}
		}
	}
	return ""
}

func (y *YT) getComments(videoID string) []string {
	var comments []string
	call := y.service.CommentThreads.List([]string{"snippet", "replies"}).VideoId(videoID).TextFormat("plainText").MaxResults(100)
	response, err := call.Do()
	if err != nil {
		log.Printf("Failed to fetch comments: %v", err)
		return comments
	}

	for _, item := range response.Items {
		topLevelComment := item.Snippet.TopLevelComment.Snippet.TextDisplay
		comments = append(comments, topLevelComment)

		if item.Replies != nil {
			for _, reply := range item.Replies.Comments {
				replyText := reply.Snippet.TextDisplay
				comments = append(comments, "    - "+replyText)
			}
		}
	}

	return comments
}

func parseDuration(durationStr string) (int, error) {
	matches := regexp.MustCompile(`(?i)PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`).FindStringSubmatch(durationStr)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration string: %s", durationStr)
	}

	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])

	return hours*60 + minutes + seconds/60, nil
}

func (y *YT) getVideoDuration(videoID string) (int, error) {
	videoResponse, err := y.service.Videos.List([]string{"contentDetails"}).Id(videoID).Do()
	if err != nil {
		return 0, fmt.Errorf("error getting video details: %v", err)
	}
	durationStr := videoResponse.Items[0].ContentDetails.Duration
	return parseDuration(durationStr)
}

type Transcript struct {
	XMLName xml.Name `xml:"transcript"`
	Texts   []Text   `xml:"text"`
}

type Text struct {
	Start string `xml:"start,attr"`
	Dur   string `xml:"dur,attr"`
	Value string `xml:",chardata"`
}

func unmarshalTranscript(xmlData []byte) (*Transcript, error) {
	var transcript Transcript
	err := xml.Unmarshal(xmlData, &transcript)
	if err != nil {
		return nil, err
	}
	return &transcript, nil
}
