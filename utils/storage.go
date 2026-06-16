package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fupcode/ImageFlow/config"
	"github.com/fupcode/ImageFlow/utils/logger"
	"go.uber.org/zap"
)

// StorageProvider defines the interface for storage operations
type StorageProvider interface {
	Store(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// LocalStorage implements StorageProvider for local filesystem
type LocalStorage struct {
	BasePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	dirs := []string{
		filepath.Join(basePath, "original", "landscape"),
		filepath.Join(basePath, "original", "portrait"),
		filepath.Join(basePath, "landscape", "webp"),
		filepath.Join(basePath, "landscape", "avif"),
		filepath.Join(basePath, "landscape", "thumb"),
		filepath.Join(basePath, "portrait", "webp"),
		filepath.Join(basePath, "portrait", "avif"),
		filepath.Join(basePath, "portrait", "thumb"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return &LocalStorage{BasePath: basePath}, nil
}

func (ls *LocalStorage) Store(ctx context.Context, key string, data []byte) error {
	fullPath := filepath.Join(ls.BasePath, key)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Failed to create directory",
			zap.String("dir", dir),
			zap.Error(err))
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		logger.Error("Failed to write file",
			zap.String("path", fullPath),
			zap.Error(err))
		return fmt.Errorf("failed to write file %s: %v", fullPath, err)
	}

	logger.Info("File stored locally",
		zap.String("key", key),
		zap.String("path", fullPath))
	return nil
}

func (ls *LocalStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(ls.BasePath, key))
}

func (ls *LocalStorage) Delete(ctx context.Context, key string) error {
	return os.Remove(filepath.Join(ls.BasePath, key))
}

// NewStorageProvider creates a new local storage provider.
func NewStorageProvider(cfg *config.Config) (StorageProvider, error) {
	return NewLocalStorage(cfg.ImageBasePath)
}

// Global storage instance
var Storage StorageProvider

// InitStorage initializes the global storage provider
func InitStorage(cfg *config.Config) error {
	var err error
	Storage, err = NewStorageProvider(cfg)
	return err
}
