package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fupcode/ImageFlow/config"
	"github.com/fupcode/ImageFlow/handlers"
	"github.com/fupcode/ImageFlow/utils"
	"github.com/fupcode/ImageFlow/utils/logger"
)

const testAPIKey = "test-api-key"

func TestBackendAPIEndpoints(t *testing.T) {
	cfg, cleanup := setupAPITest(t)
	defer cleanup()

	server := httptest.NewServer(buildTestMux(cfg))
	defer server.Close()

	client := server.Client()

	t.Run("cors preflight", func(t *testing.T) {
		req := newRequest(t, http.MethodOptions, server.URL+"/api/config", nil, false)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		if got := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
			t.Fatalf("Access-Control-Allow-Methods = %q, want GET included", got)
		}
	})

	t.Run("validate api key", func(t *testing.T) {
		req := newRequest(t, http.MethodGet, server.URL+"/api/validate-api-key", nil, false)
		resp := doRequest(t, client, req)
		resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)

		req = newRequest(t, http.MethodGet, server.URL+"/api/validate-api-key", nil, true)
		resp = doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body struct {
			Valid bool `json:"valid"`
		}
		decodeJSON(t, resp.Body, &body)
		if !body.Valid {
			t.Fatal("validate-api-key returned valid=false")
		}
	})

	t.Run("config", func(t *testing.T) {
		req := newRequest(t, http.MethodPost, server.URL+"/api/config", nil, false)
		resp := doRequest(t, client, req)
		resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)

		req = newRequest(t, http.MethodGet, server.URL+"/api/config", nil, false)
		resp = doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body config.ClientConfig
		decodeJSON(t, resp.Body, &body)
		if body.MaxUploadCount != cfg.MaxUploadCount || body.ImageQuality != cfg.ImageQuality {
			t.Fatalf("config response = %+v, want maxUploadCount=%d imageQuality=%d", body, cfg.MaxUploadCount, cfg.ImageQuality)
		}
	})

	t.Run("upload requires auth", func(t *testing.T) {
		req := newRequest(t, http.MethodPost, server.URL+"/api/upload", nil, false)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	var uploadedID string
	t.Run("upload image", func(t *testing.T) {
		body, contentType := multipartUploadBody(t, "sample.jpg", jpegBytes(t), map[string]string{
			"tags":          "nature, wallpaper",
			"expiryMinutes": "60",
		})

		req := newRequest(t, http.MethodPost, server.URL+"/api/upload", body, true)
		req.Header.Set("Content-Type", contentType)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var uploadResp struct {
			Results []handlers.UploadResult `json:"results"`
		}
		decodeJSON(t, resp.Body, &uploadResp)
		if len(uploadResp.Results) != 1 {
			t.Fatalf("upload returned %d results, want 1", len(uploadResp.Results))
		}

		result := uploadResp.Results[0]
		if result.Status != "success" {
			t.Fatalf("upload status = %q message = %q, want success", result.Status, result.Message)
		}
		if result.Orientation != "landscape" || result.Format != "jpeg" {
			t.Fatalf("upload result orientation/format = %q/%q, want landscape/jpeg", result.Orientation, result.Format)
		}
		if !containsAll(result.Tags, "nature", "wallpaper") {
			t.Fatalf("upload tags = %#v, want nature and wallpaper", result.Tags)
		}
		if result.URLs["original"] == "" || result.URLs["webp"] == "" || result.URLs["thumb"] == "" {
			t.Fatalf("upload URLs missing expected formats: %#v", result.URLs)
		}

		id := imageIDFromURL(t, result.URLs["original"])
		if id == "" {
			t.Fatal("failed to derive uploaded image ID")
		}
		uploadedID = id
	})

	if uploadedID == "" {
		t.Fatal("upload test did not record image ID")
	}

	t.Run("list images", func(t *testing.T) {
		req := newRequest(t, http.MethodGet, server.URL+"/api/images?tag=nature&orientation=landscape&format=image", nil, true)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body handlers.PaginatedResponse
		decodeJSON(t, resp.Body, &body)
		if !body.Success || body.Total != 1 || len(body.Images) != 1 {
			t.Fatalf("list response = %+v, want one image", body)
		}
		if body.Images[0].ID != uploadedID {
			t.Fatalf("listed image ID = %q, want %q", body.Images[0].ID, uploadedID)
		}
	})

	t.Run("tags", func(t *testing.T) {
		req := newRequest(t, http.MethodGet, server.URL+"/api/tags", nil, true)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body handlers.TagsResponse
		decodeJSON(t, resp.Body, &body)
		if !containsAll(body.Tags, "nature", "wallpaper") {
			t.Fatalf("tags response = %#v, want nature and wallpaper", body.Tags)
		}
	})

	t.Run("update tags", func(t *testing.T) {
		payload := strings.NewReader(`{"id":"` + uploadedID + `","tags":["city","city"," ","night"]}`)
		req := newRequest(t, http.MethodPost, server.URL+"/api/update-tags", payload, true)
		req.Header.Set("Content-Type", "application/json")
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body handlers.UpdateTagsResponse
		decodeJSON(t, resp.Body, &body)
		if !body.Success || body.ID != uploadedID {
			t.Fatalf("update-tags response = %+v, want success for %q", body, uploadedID)
		}
		if len(body.Tags) != 2 || body.Tags[0] != "city" || body.Tags[1] != "night" {
			t.Fatalf("updated tags = %#v, want [city night]", body.Tags)
		}
	})

	t.Run("random image", func(t *testing.T) {
		req := newRequest(t, http.MethodGet, server.URL+"/api/random?tag=city&orientation=landscape&format=original", nil, false)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		if got := resp.Header.Get("Content-Type"); got != "image/jpeg" {
			t.Fatalf("random Content-Type = %q, want image/jpeg", got)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			t.Fatal("random image response body is empty")
		}
	})

	t.Run("trigger cleanup", func(t *testing.T) {
		req := newRequest(t, http.MethodPost, server.URL+"/api/trigger-cleanup", nil, true)
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body map[string]string
		decodeJSON(t, resp.Body, &body)
		if body["status"] != "success" {
			t.Fatalf("trigger-cleanup response = %#v, want success", body)
		}
	})

	t.Run("delete image", func(t *testing.T) {
		payload := strings.NewReader(`{"id":"` + uploadedID + `"}`)
		req := newRequest(t, http.MethodPost, server.URL+"/api/delete-image", payload, true)
		req.Header.Set("Content-Type", "application/json")
		resp := doRequest(t, client, req)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body handlers.DeleteResponse
		decodeJSON(t, resp.Body, &body)
		if !body.Success {
			t.Fatalf("delete response = %+v, want success", body)
		}
	})
}

func setupAPITest(t *testing.T) (*config.Config, func()) {
	t.Helper()

	if err := logger.InitBasicLogger(); err != nil {
		t.Fatalf("init logger: %v", err)
	}
	if err := os.MkdirAll("logs", 0755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}

	cfg := &config.Config{
		ServerAddr:         "127.0.0.1:0",
		ImageBasePath:      t.TempDir(),
		MetadataSQLitePath: filepath.Join(t.TempDir(), "metadata.db"),
		APIKey:             testAPIKey,
		MaxUploadCount:     5,
		ImageQuality:       75,
		WorkerThreads:      1,
		WorkerPoolSize:     2,
		Speed:              5,
		CleanupInterval:    60,
	}

	if err := logger.InitLogger(cfg); err != nil {
		t.Fatalf("init full logger: %v", err)
	}

	ensureDirectories(cfg)

	if err := utils.InitStorage(cfg); err != nil {
		t.Fatalf("init storage: %v", err)
	}
	if err := utils.InitMetadataStore(cfg); err != nil {
		t.Fatalf("init metadata store: %v", err)
	}

	utils.InitVips(cfg)
	utils.InitCleaner(cfg)

	return cfg, func() {
		if utils.Cleaner != nil {
			utils.Cleaner.Stop()
		}
		if pool := utils.GetWorkerPool(); pool != nil {
			pool.Shutdown()
		}
	}
}

func buildTestMux(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/validate-api-key", handlers.ValidateAPIKey(cfg))
	mux.HandleFunc("/api/upload", handlers.RequireAPIKey(cfg, handlers.UploadHandler(cfg)))
	mux.HandleFunc("/api/images", handlers.RequireAPIKey(cfg, handlers.ListImagesHandler(cfg)))
	mux.HandleFunc("/api/delete-image", handlers.RequireAPIKey(cfg, handlers.DeleteImageHandler(cfg)))
	mux.HandleFunc("/api/update-tags", handlers.RequireAPIKey(cfg, handlers.UpdateTagsHandler(cfg)))
	mux.HandleFunc("/api/config", handlers.ConfigHandler(cfg))
	mux.HandleFunc("/api/tags", handlers.RequireAPIKey(cfg, handlers.TagsHandler(cfg)))
	mux.HandleFunc("/api/trigger-cleanup", handlers.RequireAPIKey(cfg, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		utils.TriggerCleanup()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Cleanup process triggered",
		})
	}))
	mux.HandleFunc("/api/random", handlers.LocalRandomImageHandler(cfg))
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(cfg.ImageBasePath))))

	return corsMiddleware(mux)
}

func newRequest(t *testing.T, method, url string, body io.Reader, auth bool) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+testAPIKey)
	}
	return req
}

func doRequest(t *testing.T, client *http.Client, req *http.Request) *http.Response {
	t.Helper()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d, body = %s", resp.StatusCode, want, string(data))
	}
}

func decodeJSON(t *testing.T, r io.Reader, dst any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(dst); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func multipartUploadBody(t *testing.T, filename string, data []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}

	part, err := writer.CreateFormFile("images[]", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return body, writer.FormDataContentType()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 220, G: 80, B: 40, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func imageIDFromURL(t *testing.T, rawURL string) string {
	t.Helper()
	base := filepath.Base(rawURL)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func containsAll(values []string, want ...string) bool {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	for _, value := range want {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}
