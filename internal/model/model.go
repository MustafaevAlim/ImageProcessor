package model

import "time"

type ImageInCreate struct {
	UploadsPath   string
	ProcessedPath string
	Processed     bool
}

type ImageInRepo struct {
	ID            int       `json:"id"`
	UploadsPath   string    `json:"uploads_path"`
	ProcessedPath string    `json:"processed_path"`
	Processed     bool      `json:"processed"`
	CreatedAt     time.Time `json:"created_at"`
}

type ImageTask struct {
	ImageID        int              `json:"image_id"`
	TypeProcessing string           `json:"type_processing"`
	UploadsPath    string           `json:"uploads_path"`
	Parameters     ProcessingParams `json:"parameters"`
}

type ProcessingParams struct {
	Width  *int `json:"width,omitempty"`
	Height *int `json:"height,omitempty"`

	WatermarkPath *string `json:"watermark_path,omitempty"`

	MaxSize *int `json:"max_size,omitempty"`
}
