package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fupcode/ImageFlow/utils/logger"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// SQLiteMetadataStore implements metadata storage using SQLite.
type SQLiteMetadataStore struct {
	db *sql.DB
}

// NewSQLiteMetadataStore creates a SQLite-backed metadata store.
func NewSQLiteMetadataStore(dbPath string) (*SQLiteMetadataStore, error) {
	if dbPath == "" {
		dbPath = "data/metadata.db"
	}
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(".", dbPath)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %v", err)
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteMetadataStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	logger.Info("SQLite metadata store initialized", zap.String("path", dbPath))
	return store, nil
}

func (sms *SQLiteMetadataStore) initSchema(ctx context.Context) error {
	_, err := sms.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS image_metadata (
	id TEXT PRIMARY KEY,
	original_name TEXT NOT NULL,
	upload_time TEXT NOT NULL,
	expiry_time TEXT,
	format TEXT NOT NULL,
	orientation TEXT NOT NULL,
	tags_json TEXT NOT NULL,
	paths_json TEXT NOT NULL,
	sizes_json TEXT NOT NULL,
	metadata_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_image_metadata_upload_time ON image_metadata(upload_time);
CREATE INDEX IF NOT EXISTS idx_image_metadata_expiry_time ON image_metadata(expiry_time);
CREATE INDEX IF NOT EXISTS idx_image_metadata_orientation ON image_metadata(orientation);
`)
	if err != nil {
		return fmt.Errorf("failed to initialize sqlite schema: %v", err)
	}
	return nil
}

// SaveMetadata saves image metadata to SQLite.
func (sms *SQLiteMetadataStore) SaveMetadata(ctx context.Context, metadata *ImageMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}
	tagsJSON, err := json.Marshal(metadata.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %v", err)
	}
	pathsJSON, err := json.Marshal(metadata.Paths)
	if err != nil {
		return fmt.Errorf("failed to marshal paths: %v", err)
	}
	sizesJSON, err := json.Marshal(metadata.Sizes)
	if err != nil {
		return fmt.Errorf("failed to marshal sizes: %v", err)
	}

	expiryTime := ""
	if !metadata.ExpiryTime.IsZero() {
		expiryTime = metadata.ExpiryTime.UTC().Format(time.RFC3339)
	}

	_, err = sms.db.ExecContext(ctx, `
INSERT INTO image_metadata (
	id, original_name, upload_time, expiry_time, format, orientation,
	tags_json, paths_json, sizes_json, metadata_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	original_name=excluded.original_name,
	upload_time=excluded.upload_time,
	expiry_time=excluded.expiry_time,
	format=excluded.format,
	orientation=excluded.orientation,
	tags_json=excluded.tags_json,
	paths_json=excluded.paths_json,
	sizes_json=excluded.sizes_json,
	metadata_json=excluded.metadata_json
`, metadata.ID, metadata.OriginalName, metadata.UploadTime.UTC().Format(time.RFC3339), expiryTime,
		metadata.Format, metadata.Orientation, string(tagsJSON), string(pathsJSON), string(sizesJSON), string(metadataJSON))
	if err != nil {
		return fmt.Errorf("failed to save metadata to SQLite: %v", err)
	}

	return nil
}

// GetMetadata retrieves image metadata from SQLite.
func (sms *SQLiteMetadataStore) GetMetadata(ctx context.Context, id string) (*ImageMetadata, error) {
	var data string
	err := sms.db.QueryRowContext(ctx, `SELECT metadata_json FROM image_metadata WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("metadata not found for ID: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from SQLite: %v", err)
	}

	var metadata ImageMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}
	return &metadata, nil
}

// ListExpiredImages lists all expired images.
func (sms *SQLiteMetadataStore) ListExpiredImages(ctx context.Context) ([]*ImageMetadata, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := sms.db.QueryContext(ctx, `
SELECT metadata_json FROM image_metadata
WHERE expiry_time IS NOT NULL AND expiry_time != '' AND expiry_time <= ?
`, now)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired images from SQLite: %v", err)
	}
	defer rows.Close()

	return scanMetadataRows(rows)
}

// DeleteMetadata deletes image metadata from SQLite.
func (sms *SQLiteMetadataStore) DeleteMetadata(ctx context.Context, id string) error {
	_, err := sms.db.ExecContext(ctx, `DELETE FROM image_metadata WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete metadata from SQLite: %v", err)
	}
	return nil
}

// GetAllMetadata retrieves all image metadata from SQLite.
func (sms *SQLiteMetadataStore) GetAllMetadata(ctx context.Context) ([]*ImageMetadata, error) {
	rows, err := sms.db.QueryContext(ctx, `SELECT metadata_json FROM image_metadata ORDER BY upload_time DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata from SQLite: %v", err)
	}
	defer rows.Close()

	allMetadata, err := scanMetadataRows(rows)
	if err != nil {
		return nil, err
	}

	logger.Info("Retrieved all metadata entries from SQLite", zap.Int("count", len(allMetadata)))
	return allMetadata, nil
}

func scanMetadataRows(rows *sql.Rows) ([]*ImageMetadata, error) {
	var result []*ImageMetadata
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan metadata row: %v", err)
		}

		var metadata ImageMetadata
		if err := json.Unmarshal([]byte(data), &metadata); err != nil {
			logger.Warn("Failed to unmarshal SQLite metadata", zap.Error(err))
			continue
		}
		result = append(result, &metadata)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read metadata rows: %v", err)
	}
	return result, nil
}

func (sms *SQLiteMetadataStore) IsEmpty(ctx context.Context) (bool, error) {
	var count int
	if err := sms.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM image_metadata`).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to count SQLite metadata rows: %v", err)
	}
	return count == 0, nil
}
