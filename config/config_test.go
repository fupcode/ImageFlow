package config

import "testing"

func TestGetBaseURLUsesCustomDomainForLocalStorage(t *testing.T) {
	cfg := &Config{
		CustomDomain: "https://img.fupengcheng.top/",
	}

	got := cfg.GetBaseURL()
	want := "https://img.fupengcheng.top/images"
	if got != want {
		t.Fatalf("GetBaseURL() = %q, want %q", got, want)
	}
}

func TestGetBaseURLNormalizesLocalhostCustomDomain(t *testing.T) {
	cfg := &Config{
		CustomDomain: "localhost:8000",
	}

	got := cfg.GetBaseURL()
	want := "http://localhost:8000/images"
	if got != want {
		t.Fatalf("GetBaseURL() = %q, want %q", got, want)
	}
}

func TestGetBaseURLNormalizesBareCustomDomain(t *testing.T) {
	cfg := &Config{
		CustomDomain: "img.example.com/",
	}

	got := cfg.GetBaseURL()
	want := "https://img.example.com/images"
	if got != want {
		t.Fatalf("GetBaseURL() = %q, want %q", got, want)
	}
}

func TestGetBaseURLDefaultsToRelativeImagesForLocalStorage(t *testing.T) {
	cfg := &Config{}

	got := cfg.GetBaseURL()
	want := "/images"
	if got != want {
		t.Fatalf("GetBaseURL() = %q, want %q", got, want)
	}
}

func TestLoadReadsAvifEnabled(t *testing.T) {
	t.Setenv("AVIF_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AvifSupport {
		t.Fatal("AvifSupport = true, want false")
	}
}

func TestLoadDisablesAvifByDefault(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AvifSupport {
		t.Fatal("AvifSupport = true, want false by default")
	}
}
