package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func main() {
	// Load environment variables from .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get API key from environment variables
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		log.Fatal("YouTube API key not found in environment")
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.GET("/", showHomePage)
	router.POST("/search", handleSearch(apiKey))
	router.GET("/embed/:id", embedVideo)

	router.Run(":8080")
}

// Show the home page with the search form
func showHomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

// Handle search and return top 10 video results
func handleSearch(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.PostForm("query")

		videos := searchYouTube(apiKey, query)
		if len(videos) == 0 {
			c.String(http.StatusNotFound, "No videos found.")
			return
		}

		c.HTML(http.StatusOK, "results.html", gin.H{"Videos": videos})
	}
}

// Search YouTube using the API key and return video details
func searchYouTube(apiKey, query string) []map[string]string {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		fmt.Println("Error initializing YouTube service:", err)
		return nil
	}

	// Search for the top 10 videos based on the query
	searchCall := service.Search.List([]string{"id", "snippet"}).Q(query).MaxResults(10).Type("video")
	searchResponse, err := searchCall.Do()
	if err != nil {
		fmt.Println("Error searching YouTube:", err)
		return nil
	}

	// Collect video IDs for content details request
	var videoIDs []string
	for _, item := range searchResponse.Items {
		videoIDs = append(videoIDs, item.Id.VideoId)
	}

	// Fetch additional details (like duration) using the video IDs
	detailsCall := service.Videos.List([]string{"snippet", "contentDetails"}).Id(strings.Join(videoIDs, ","))
	detailsResponse, err := detailsCall.Do()
	if err != nil {
		fmt.Println("Error fetching video details:", err)
		return nil
	}

	videos := make([]map[string]string, 0, len(detailsResponse.Items))
	for _, item := range detailsResponse.Items {
		video := map[string]string{
			"id":        item.Id,
			"title":     item.Snippet.Title,
			"channel":   item.Snippet.ChannelTitle,
			"duration":  formatDuration(item.ContentDetails.Duration),
		}
		videos = append(videos, video)
	}

	return videos
}

// Format ISO 8601 duration to H:MM:SS or MM:SS
func formatDuration(duration string) string {
	d, _ := time.ParseDuration(strings.ReplaceAll(strings.ToLower(duration), "pt", ""))

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}


// Embed the selected video
func embedVideo(c *gin.Context) {
	videoID := c.Param("id")
	embedURL := fmt.Sprintf("https://www.youtube.com/embed/%s", videoID)
	c.HTML(http.StatusOK, "embed.html", gin.H{"EmbedURL": embedURL})
}
