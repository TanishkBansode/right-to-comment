package database

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS comments (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            video_id TEXT NOT NULL,
            comment TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
	)
	if err != nil {
		return err
	}
	return nil
}

func AddComment(videoId, commentText string) error {
	_, err := db.ExecContext(
		context.Background(),
		"INSERT INTO comments (video_id, comment) VALUES (?, ?)",
		videoId, commentText,
	)
	return err
}

func GetComments(videoId string) ([]map[string]string, error) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT comment, created_at FROM comments WHERE video_id = ? ORDER BY created_at DESC",
		videoId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []map[string]string
	for rows.Next() {
		var comment, createdAt string
		if err := rows.Scan(&comment, &createdAt); err != nil {
			return nil, err
		}
		comments = append(comments, map[string]string{
			"text":      comment,
			"createdAt": createdAt,
		})
	}
	return comments, nil
}
