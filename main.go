package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TanishkBansode/right-to-comment/database"

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
	database.InitDB("./data")

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	router.GET("/comments/:videoId", getComments)
	router.POST("/comments/:videoId", addComment)
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
			"id":       item.Id,
			"title":    item.Snippet.Title,
			"channel":  item.Snippet.ChannelTitle,
			"duration": formatDuration(item.ContentDetails.Duration),
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
	c.HTML(http.StatusOK, "embed.html", gin.H{"EmbedURL": embedURL, "VideoID": videoID})
}

func addComment(c *gin.Context) {
	videoId := c.Param("videoId")
	commentText := c.PostForm("comment")

	err := database.AddComment(videoId, commentText)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_template.html", gin.H{"error": "Failed to add comment"})
		return
	}

	// Fetch updated comments after adding the new one
	getComments(c)
}

func getComments(c *gin.Context) {
	videoId := c.Param("videoId")
	comments, err := database.GetComments(videoId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load comments"})
		return
	}

	// Create a string builder for comments HTML
	var commentsHTML strings.Builder
	for _, comment := range comments {
		commentsHTML.WriteString(fmt.Sprintf("<div><p><strong>%s</strong>: %s</p></div>", comment["createdAt"], comment["text"]))
	}

	c.Data(http.StatusOK, "text/html", []byte(commentsHTML.String()))
}
