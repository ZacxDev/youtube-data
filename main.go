package fetcher

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type VideoPost struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	EmbedURL string `json:"embedUrl"`
	URL      string `json:"url"`
}

type Response struct {
	Posts   []VideoPost `json:"posts"`
	HasMore bool        `json:"hasMore"`
}

type YouTubeFetcher struct {
	apiKey  string
	service *youtube.Service
}

func NewYouTubeFetcher(apiKey string) (*YouTubeFetcher, error) {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error creating YouTube client: %v", err)
	}

	return &YouTubeFetcher{
		apiKey:  apiKey,
		service: service,
	}, nil
}

func (f *YouTubeFetcher) FetchVideos(channelID string, page int) (*Response, error) {
	if page < 1 {
		return nil, fmt.Errorf("page number must be greater than 0")
	}

	// Get the uploads playlist ID for the channel
	channelResponse, err := f.service.Channels.List([]string{"contentDetails"}).
		Id(channelID).
		Do()
	if err != nil {
		return nil, fmt.Errorf("error getting channel details: %v", err)
	}
	if len(channelResponse.Items) == 0 {
		return nil, fmt.Errorf("channel not found")
	}

	uploadsPlaylistID := channelResponse.Items[0].ContentDetails.RelatedPlaylists.Uploads
	response := &Response{
		Posts:   make([]VideoPost, 0),
		HasMore: false,
	}

	// Calculate the page token needed for the requested page
	var nextPageToken string
	for i := 1; i < page; i++ {
		tempResponse, err := f.service.PlaylistItems.List([]string{"snippet"}).
			PlaylistId(uploadsPlaylistID).
			MaxResults(50).
			PageToken(nextPageToken).
			Do()
		if err != nil {
			return nil, fmt.Errorf("error fetching playlist items: %v", err)
		}

		nextPageToken = tempResponse.NextPageToken
		if nextPageToken == "" {
			return nil, fmt.Errorf("page %d is beyond the available results", page)
		}
	}

	// Fetch the requested page
	playlistResponse, err := f.service.PlaylistItems.List([]string{"snippet"}).
		PlaylistId(uploadsPlaylistID).
		MaxResults(50).
		PageToken(nextPageToken).
		Do()
	if err != nil {
		return nil, fmt.Errorf("error fetching playlist items: %v", err)
	}

	// Process each video in the response
	for _, item := range playlistResponse.Items {
		video := item.Snippet
		videoID := video.ResourceId.VideoId
		post := VideoPost{
			ID:       videoID,
			Title:    video.Title,
			EmbedURL: fmt.Sprintf("https://www.youtube.com/embed/%s", videoID),
			URL:      fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		}
		response.Posts = append(response.Posts, post)
	}

	response.HasMore = playlistResponse.NextPageToken != ""
	return response, nil
}
