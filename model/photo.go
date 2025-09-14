package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PhotoDB struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	LonLat        *GeoPoint           `bson:"lonlat,omitempty"`
	TakenAt       time.Time          `bson:"taken_at,omitempty"`
	FilePath      string             `bson:"file_path"`
	ThumbnailPath string             `bson:"thumbnail_path,omitempty"`
	Metadata      map[string]any     `bson:"metadata,omitempty"`
	Size          int64              `bson:"size"`
	ContentType   string             `bson:"content_type"`
}

type GeoPoint struct {
	Type        string    `bson:"type,omitempty"`
	Coordinates []float64 `bson:"coordinates,omitempty"` // [longitude, latitude]
}
