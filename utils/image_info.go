package utils

// ImageInfo represents information about an image.
type ImageInfo struct {
	ID          string            `json:"id"`
	FileName    string            `json:"filename"`
	URL         string            `json:"url"`
	URLs        map[string]string `json:"urls"`
	Orientation string            `json:"orientation"`
	Format      string            `json:"format"`
	Size        int64             `json:"size"`
	Path        string            `json:"path"`
	StorageType string            `json:"storageType"`
	Tags        []string          `json:"tags"`
}
