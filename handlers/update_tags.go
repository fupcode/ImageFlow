package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fupcode/ImageFlow/config"
	"github.com/fupcode/ImageFlow/utils"
	"github.com/fupcode/ImageFlow/utils/errors"
	"github.com/fupcode/ImageFlow/utils/logger"
	"go.uber.org/zap"
)

// UpdateTagsRequest represents the request body for updating image tags.
type UpdateTagsRequest struct {
	ID   string   `json:"id"`
	Tags []string `json:"tags"`
}

// UpdateTagsResponse represents the response after updating image tags.
type UpdateTagsResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
}

// UpdateTagsHandler returns a handler for updating image tags.
func UpdateTagsHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errors.HandleError(w, errors.ErrInvalidParam, "Method not allowed", nil)
			logger.Warn("Invalid request method",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path))
			return
		}

		var req UpdateTagsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errors.HandleError(w, errors.ErrInvalidParam, "Invalid request body", nil)
			logger.Warn("Failed to decode update tags request body",
				zap.Error(err))
			return
		}

		req.ID = strings.TrimSpace(req.ID)
		if req.ID == "" {
			errors.HandleError(w, errors.ErrInvalidParam, "Image ID is required", nil)
			logger.Warn("Missing image ID for tag update")
			return
		}

		tags := normalizeTags(req.Tags)
		metadata, err := utils.UpdateMetadataTags(r.Context(), req.ID, tags)
		if err != nil {
			logger.Warn("Failed to update image tags",
				zap.String("image_id", req.ID),
				zap.Error(err))
			errors.HandleError(w, errors.ErrMetadata, "Failed to update image tags", nil)
			return
		}

		resp := UpdateTagsResponse{
			Success: true,
			Message: "Tags updated successfully",
			ID:      metadata.ID,
			Tags:    metadata.Tags,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logger.Error("Failed to encode update tags response",
				zap.String("image_id", req.ID),
				zap.Error(err))
			errors.HandleError(w, errors.ErrInternal, "Internal server error", nil)
			return
		}

		logger.Info("Image tags updated",
			zap.String("image_id", metadata.ID),
			zap.Strings("tags", metadata.Tags),
			zap.String("storage_type", "local"))
	}
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}

		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}

	return normalized
}
