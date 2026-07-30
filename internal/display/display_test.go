package display

import (
	"strings"
	"testing"
)

func TestBannerWithoutVersion(t *testing.T) {
	SetVersion("")
	b := Banner()
	if !strings.Contains(b, "Archcore") {
		t.Fatalf("Banner missing 'Archcore': %q", b)
	}
	if strings.Contains(b, "v0.5.4") {
		t.Fatalf("Banner should not contain version when unset: %q", b)
	}
}

func TestBannerWithVersion(t *testing.T) {
	SetVersion("v0.5.4")
	b := Banner()
	if !strings.Contains(b, "v0.5.4") {
		t.Fatalf("Banner missing version 'v0.5.4': %q", b)
	}
}

func TestBannerWithDevVersion(t *testing.T) {
	SetVersion("dev")
	b := Banner()
	if strings.Contains(b, "dev") {
		t.Fatalf("Banner should not contain 'dev' version: %q", b)
	}
}

func TestWelcomeBannerWithoutVersion(t *testing.T) {
	SetVersion("")
	wb := WelcomeBanner()
	if !strings.Contains(wb, "Archcore") {
		t.Fatalf("WelcomeBanner missing 'Archcore': %q", wb)
	}
	if strings.Contains(wb, "v0.5.4") {
		t.Fatalf("WelcomeBanner should not contain version when unset: %q", wb)
	}
}

func TestWelcomeBannerWithVersion(t *testing.T) {
	SetVersion("v0.5.4")
	wb := WelcomeBanner()
	if !strings.Contains(wb, "v0.5.4") {
		t.Fatalf("WelcomeBanner missing version 'v0.5.4': %q", wb)
	}
}

func TestWelcomeBannerWithDevVersion(t *testing.T) {
	SetVersion("dev")
	wb := WelcomeBanner()
	if strings.Contains(wb, "dev") {
		t.Fatalf("WelcomeBanner should not contain 'dev' version: %q", wb)
	}
}

func TestVersionSuffixEmpty(t *testing.T) {
	SetVersion("")
	got := versionSuffix()
	if got != "" {
		t.Fatalf("expected empty suffix, got %q", got)
	}
}

func TestVersionSuffixDev(t *testing.T) {
	SetVersion("dev")
	got := versionSuffix()
	if got != "" {
		t.Fatalf("expected empty suffix for dev, got %q", got)
	}
}

func TestVersionSuffixReal(t *testing.T) {
	SetVersion("v1.2.3")
	got := versionSuffix()
	if got != " v1.2.3" {
		t.Fatalf("expected ' v1.2.3', got %q", got)
	}
}
