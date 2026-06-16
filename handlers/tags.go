package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/fupcode/ImageFlow/config"
	"github.com/fupcode/ImageFlow/utils"
	"github.com/fupcode/ImageFlow/utils/errors"
	"github.com/fupcode/ImageFlow/utils/logger"
	"go.uber.org/zap"
)

// TagsResponse represents the response for the tags API
type TagsResponse struct {
	Tags []string `json:"tags"`
}

// TagsHandler returns a handler for retrieving all unique tags
func TagsHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Processing tags request")

		tags, err := getAllUniqueTags()
		if err != nil {
			logger.Error("Failed to retrieve tags", zap.Error(err))
			errors.HandleError(w, errors.ErrInternal, "Failed to retrieve tags", err)
			return
		}

		logger.Debug("Retrieved unique tags",
			zap.Int("count", len(tags)))

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(TagsResponse{Tags: tags}); err != nil {
			logger.Error("Failed to encode tags response", zap.Error(err))
			errors.HandleError(w, errors.ErrInternal, "Failed to encode response", err)
			return
		}
	}
}

// getAllUniqueTags retrieves all unique tags from image metadata
func getAllUniqueTags() ([]string, error) {
	logger.Debug("Using metadata store to get unique tags")

	uniqueTags := make(map[string]struct{})
	allMetadata, err := utils.MetadataManager.GetAllMetadata(context.Background())
	if err != nil {
		return nil, err
	}
	for _, metadata := range allMetadata {
		for _, tag := range metadata.Tags {
			uniqueTags[tag] = struct{}{}
		}
	}

	logger.Debug("Processed metadata entries",
		zap.Int("total_entries", len(allMetadata)),
		zap.Int("unique_tags", len(uniqueTags)))

	return mapKeysToSortedSlice(uniqueTags), nil
}

// mapKeysToSortedSlice converts map keys to a sorted slice
func mapKeysToSortedSlice(m map[string]struct{}) []string {
	result := make([]string, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
