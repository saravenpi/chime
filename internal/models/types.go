package models

import "time"

type Chat struct {
	ROWID        int64
	ChatID       string
	DisplayName  string
	LastMessage  string
	LastTime     time.Time
	UnreadCount  int
	Participants []string
	IsGroup      bool
	HasUnread    bool
}

type Message struct {
	ROWID          int64
	GUID           string
	Text           string
	Handle         string
	IsFromMe       bool
	Date           time.Time
	ChatID         int64
	AttachmentPath string
}

type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
)
