package model

type Photo struct {
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	FileContent string `json:"file_content"`
}
