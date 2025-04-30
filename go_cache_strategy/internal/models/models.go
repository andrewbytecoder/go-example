package models

import "time"

// 定义数据结构

// User represents a user data structure.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LogEntry represents a log data structure for Write-Around example.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	ID        string    `json:"id"`
}
