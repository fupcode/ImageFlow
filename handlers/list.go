package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fupcode/ImageFlow/config"
	"github.com/fupcode/ImageFlow/utils"
	"github.com/fupcode/ImageFlow/utils/errors"
	"github.com/fupcode/ImageFlow/utils/logger"
	"go.uber.org/zap"
)

// ImageInfo represents information about an image
type ImageInfo = utils.ImageInfo

// PaginatedResponse represents a paginated response with images
type PaginatedResponse struct {
	Success    bool        `json:"success"`    // Whether the request was successful
	Images     []ImageInfo `json:"images"`     // Images for current page
	Page       int         `json:"page"`       // Current page number
	Limit      int         `json:"limit"`      // Number of items per page
	TotalPages int         `json:"totalPages"` // Total number of pages
	Total      int         `json:"total"`      // Total number of images
}

// ListImagesHandler returns a handler for listing images
func ListImagesHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		defer func() {
			if cfg.DebugMode {
				duration := time.Since(startTime)
				logger.Debug("List API latency",
					zap.Duration("duration", duration))
			}
		}()

		// Parse query parameters
		params := parseQueryParams(r)
		if params.format == "avif" && !cfg.AvifSupport {
			params.format = "webp"
		}

		allImages, err := listImagesFromMetadata(r.Context(), params, cfg)
		if err != nil {
			logger.Error("Failed to list images from metadata store", zap.Error(err))
			errors.HandleError(w, errors.ErrImageList, "Failed to retrieve image list", err)
			return
		}

		// Calculate pagination values
		total := len(allImages)
		totalPages := int(math.Ceil(float64(total) / float64(params.limit)))

		// Ensure page is within valid range
		if params.page > totalPages && totalPages > 0 {
			params.page = totalPages
		}

		// Calculate start and end indices for the current page
		startIdx := (params.page - 1) * params.limit
		endIdx := startIdx + params.limit
		if endIdx > total {
			endIdx = total
		}

		// Extract the subset of images for the current page
		var pagedImages []ImageInfo
		if startIdx < total {
			pagedImages = allImages[startIdx:endIdx]
		} else {
			pagedImages = []ImageInfo{}
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := PaginatedResponse{
			Success:    true,
			Images:     pagedImages,
			Page:       params.page,
			Limit:      params.limit,
			TotalPages: totalPages,
			Total:      total,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			if cfg.DebugMode {
				logger.Debug("Error encoding JSON response", zap.Error(err))
			}
		}
	}
}

// Query parameters structure
type queryParams struct {
	orientation string
	format      string
	tag         string // Tag to filter by
	page        int
	limit       int
}

// parseQueryParams extracts and validates query parameters
func parseQueryParams(r *http.Request) queryParams {
	orientation := r.URL.Query().Get("orientation")
	format := r.URL.Query().Get("format")
	tag := r.URL.Query().Get("tag")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Default values
	if orientation == "" {
		orientation = "all" // all, landscape, portrait
	}
	if format == "" {
		format = "all" // all, image, gif
	}
	// Tag can be empty, which means no tag filtering

	// Set default pagination values
	page := 1
	limit := 12 // Default items per page

	// Parse page number
	if pageStr != "" {
		pageVal, err := strconv.Atoi(pageStr)
		if err == nil && pageVal > 0 {
			page = pageVal
		}
	}

	// Parse limit
	if limitStr != "" {
		limitVal, err := strconv.Atoi(limitStr)
		if err == nil && limitVal > 0 && limitVal <= 50 { // Cap at 50 items per page
			limit = limitVal
		}
	}

	return queryParams{
		orientation: orientation,
		format:      format,
		tag:         tag,
		page:        page,
		limit:       limit,
	}
}

// listImagesFromMetadata retrieves images from the configured metadata store.
func listImagesFromMetadata(ctx context.Context, params queryParams, cfg *config.Config) ([]ImageInfo, error) {
	allMetadata, err := utils.MetadataManager.GetAllMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %v", err)
	}

	images := make([]ImageInfo, 0, len(allMetadata))
	for _, metadata := range allMetadata {
		if params.tag != "" && !metadataHasTag(metadata, params.tag) {
			continue
		}
		if params.orientation != "all" && metadata.Orientation != params.orientation {
			continue
		}

		isGIF := metadata.Format == "gif"
		if !matchesFormatFilter(params.format, isGIF) {
			continue
		}
		displayFormat := displayFormatForFilter(params.format, isGIF, cfg)

		imageInfo := ImageInfo{
			ID:          metadata.ID,
			FileName:    metadata.OriginalName,
			Orientation: metadata.Orientation,
			Format:      metadata.Format,
			StorageType: "local",
			Tags:        metadata.Tags,
			URLs:        make(map[string]string, 4),
		}

		// Get base URL for image access
		baseURL := cfg.GetBaseURL()

		// Construct URLs based on paths
		if isGIF {
			gifPath := filepath.Join("gif", metadata.ID+".gif")
			gifURL := fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(gifPath, "\\", "/"))
			imageInfo.URLs["original"] = gifURL
			imageInfo.URLs["webp"] = gifURL
			imageInfo.URLs["gif"] = gifURL
			if cfg.AvifSupport {
				imageInfo.URLs["avif"] = gifURL
			}
		} else {
			// Use stored paths if available
			if metadata.Paths.Original != "" {
				imageInfo.URLs["original"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(metadata.Paths.Original, "\\", "/"))
			} else {
				originalPath := filepath.Join("original", metadata.Orientation, metadata.ID+"."+metadata.Format)
				imageInfo.URLs["original"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(originalPath, "\\", "/"))
			}

			if metadata.Paths.WebP != "" {
				imageInfo.URLs["webp"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(metadata.Paths.WebP, "\\", "/"))
			} else {
				webpPath := filepath.Join(metadata.Orientation, "webp", metadata.ID+".webp")
				imageInfo.URLs["webp"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(webpPath, "\\", "/"))
			}

			if cfg.AvifSupport {
				if metadata.Paths.AVIF != "" {
					imageInfo.URLs["avif"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(metadata.Paths.AVIF, "\\", "/"))
				} else {
					avifPath := filepath.Join(metadata.Orientation, "avif", metadata.ID+".avif")
					imageInfo.URLs["avif"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(avifPath, "\\", "/"))
				}
			}

			if metadata.Paths.Thumb != "" {
				imageInfo.URLs["thumb"] = fmt.Sprintf("%s/%s", baseURL, strings.ReplaceAll(metadata.Paths.Thumb, "\\", "/"))
			}
		}

		// Set the requested format URL
		imageInfo.URL = imageInfo.URLs[displayFormat]
		if imageInfo.URL == "" {
			imageInfo.URL = imageInfo.URLs["original"]
		}

		// Update filename based on format
		if !isGIF && displayFormat != "original" {
			baseName := strings.TrimSuffix(imageInfo.FileName, filepath.Ext(imageInfo.FileName))
			imageInfo.FileName = baseName + "." + displayFormat
		}

		if size, exists := metadata.Sizes[displayFormat]; exists && size > 0 {
			imageInfo.Size = size
		}

		images = append(images, imageInfo)
	}

	// Sort by filename in descending order
	sort.Slice(images, func(i, j int) bool {
		return images[i].FileName > images[j].FileName
	})

	return images, nil
}

func metadataHasTag(metadata *utils.ImageMetadata, tag string) bool {
	for _, imageTag := range metadata.Tags {
		if imageTag == tag {
			return true
		}
	}
	return false
}

func matchesFormatFilter(format string, isGIF bool) bool {
	switch format {
	case "gif":
		return isGIF
	case "image", "webp", "avif", "original":
		return !isGIF
	case "all":
		return true
	default:
		return true
	}
}

func displayFormatForFilter(format string, isGIF bool, cfg *config.Config) string {
	if isGIF {
		return "original"
	}

	switch format {
	case "original":
		return "original"
	case "avif":
		if cfg.AvifSupport {
			return "avif"
		}
		return "webp"
	default:
		return "webp"
	}
}
