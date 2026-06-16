package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/fupcode/ImageFlow/config"
)

// ImageMetadata stores metadata information for images.
type ImageMetadata struct {
	ID           string           `json:"id"`
	OriginalName string           `json:"originalName"`
	UploadTime   time.Time        `json:"uploadTime"`
	ExpiryTime   time.Time        `json:"expiryTime"`
	Format       string           `json:"format"`
	Orientation  string           `json:"orientation"`
	Tags         []string         `json:"tags"`
	Sizes        map[string]int64 `json:"sizes"`
	Paths        struct {
		Original string `json:"original"`
		WebP     string `json:"webp"`
		AVIF     string `json:"avif"`
		Thumb    string `json:"thumb"`
	} `json:"paths"`
}

// MetadataStore defines the interface for metadata storage operations.
type MetadataStore interface {
	SaveMetadata(ctx context.Context, metadata *ImageMetadata) error
	GetMetadata(ctx context.Context, id string) (*ImageMetadata, error)
	ListExpiredImages(ctx context.Context) ([]*ImageMetadata, error)
	DeleteMetadata(ctx context.Context, id string) error
	GetAllMetadata(ctx context.Context) ([]*ImageMetadata, error)
}

// Global metadata storage instance.
var MetadataManager MetadataStore

// InitMetadataStore initializes SQLite metadata storage.
func InitMetadataStore(cfg *config.Config) error {
	sqliteStore, err := NewSQLiteMetadataStore(cfg.MetadataSQLitePath)
	if err != nil {
		return fmt.Errorf("failed to create SQLite metadata store: %v", err)
	}
	MetadataManager = sqliteStore
	return nil
}

// UpdateMetadataTags updates the tags for an existing image.
func UpdateMetadataTags(ctx context.Context, id string, tags []string) (*ImageMetadata, error) {
	metadata, err := MetadataManager.GetMetadata(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %v", err)
	}

	metadata.Tags = tags

	if err := MetadataManager.SaveMetadata(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %v", err)
	}

	return metadata, nil
}
