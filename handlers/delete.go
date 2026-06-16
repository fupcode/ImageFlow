package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fupcode/ImageFlow/config"
	"github.com/fupcode/ImageFlow/utils"
	"github.com/fupcode/ImageFlow/utils/errors"
	"github.com/fupcode/ImageFlow/utils/logger"
	"go.uber.org/zap"
)

// DeleteRequest represents the request body for deleting an image
type DeleteRequest struct {
	ID string `json:"id"` // Image ID (filename without extension)
}

// DeleteResponse represents the response after deleting an image
type DeleteResponse struct {
	Success bool   `json:"success"` // Whether the operation was successful
	Message string `json:"message"` // Description of the result
}

// DeleteImageHandler returns a handler for deleting images
func DeleteImageHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST method
		if r.Method != http.MethodPost {
			errors.HandleError(w, errors.ErrInvalidParam, "Method not allowed", nil)
			logger.Warn("Invalid request method",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path))
			return
		}

		// Parse the request body
		var req DeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errors.HandleError(w, errors.ErrInvalidParam, "Invalid request body", nil)
			logger.Warn("Failed to decode request body",
				zap.Error(err))
			return
		}

		// Check if ID is provided
		if req.ID == "" {
			errors.HandleError(w, errors.ErrInvalidParam, "Image ID is required", nil)
			logger.Warn("Missing image ID")
			return
		}

		logger.Info("Processing delete request",
			zap.String("image_id", req.ID))

		success, message := deleteLocalImages(req.ID, cfg.ImageBasePath)

		// If deletion was successful, clean up metadata
		if success && utils.MetadataManager != nil {
			if err := utils.MetadataManager.DeleteMetadata(r.Context(), req.ID); err != nil {
				logger.Warn("Failed to delete metadata",
					zap.String("image_id", req.ID),
					zap.Error(err))
			}
		}

		// Prepare and send response
		resp := DeleteResponse{
			Success: success,
			Message: message,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logger.Error("Failed to encode response",
				zap.String("image_id", req.ID),
				zap.Error(err))
			errors.HandleError(w, errors.ErrInternal, "Internal server error", nil)
			return
		}

		logger.Info("Delete operation completed",
			zap.String("image_id", req.ID),
			zap.Bool("success", success),
			zap.String("message", message))
	}
}

// deleteLocalImages deletes all formats of an image from local storage
func deleteLocalImages(id string, basePath string) (bool, string) {
	// Formats and orientations to check for image files
	formats := []string{"original", "webp", "avif", "thumb"}
	orientations := []string{"landscape", "portrait"}

	deletedCount := 0
	errorCount := 0
	var lastError error

	// Find all matching image files and delete them
	for _, format := range formats {
		for _, orientation := range orientations {
			var path string
			if format == "original" {
				path = filepath.Join(basePath, format, orientation)
			} else {
				path = filepath.Join(basePath, orientation, format)
			}

			// Find matching files with glob pattern
			files, err := filepath.Glob(filepath.Join(path, id+".*"))
			if err != nil {
				logger.Error("Failed to find files",
					zap.String("image_id", id),
					zap.String("path", path),
					zap.Error(err))
				errorCount++
				lastError = err
				continue
			}

			// Delete each found file
			for _, file := range files {
				err := os.Remove(file)
				if err != nil {
					logger.Error("Failed to delete file",
						zap.String("file", file),
						zap.Error(err))
					errorCount++
					lastError = err
				} else {
					logger.Debug("Successfully deleted file",
						zap.String("file", file))
					deletedCount++
				}
			}
		}
	}

	// Check for GIF files
	gifPath := filepath.Join(basePath, "gif")
	gifFiles, err := filepath.Glob(filepath.Join(gifPath, id+".*"))
	if err != nil {
		logger.Error("Failed to find GIF files",
			zap.String("image_id", id),
			zap.String("path", gifPath),
			zap.Error(err))
		errorCount++
		lastError = err
	} else {
		// Delete each GIF file found
		for _, file := range gifFiles {
			err := os.Remove(file)
			if err != nil {
				logger.Error("Failed to delete GIF file",
					zap.String("file", file),
					zap.Error(err))
				errorCount++
				lastError = err
			} else {
				logger.Debug("Successfully deleted GIF file",
					zap.String("file", file))
				deletedCount++
			}
		}
	}

	// Determine operation result
	if errorCount > 0 {
		return false, fmt.Sprintf("Partial deletion failure: %d files deleted successfully, %d failed: %v",
			deletedCount, errorCount, lastError)
	}

	if deletedCount == 0 {
		return false, "No matching image files found"
	}

	return true, fmt.Sprintf("Successfully deleted %d images", deletedCount)
}
