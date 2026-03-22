package models

import "time"

type Calendar struct {
	ID           int
	Code         string
	Name         string
	ICS_url      string
	ETag         *string
	LastModified *string
	ContentHash  *string
	LastChecked  *time.Time
}
