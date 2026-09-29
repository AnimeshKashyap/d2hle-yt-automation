package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateLocalPath(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "media")
	if err := os.MkdirAll(media, 0o700); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(media, "clip.mp4")
	if err := os.WriteFile(good, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	abs, err := ValidateLocalPath(good, []string{media})
	if err != nil {
		t.Fatal(err)
	}
	if abs != good {
		t.Fatalf("expected %s got %s", good, abs)
	}

	empty := filepath.Join(media, "empty.mp4")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateLocalPath(empty, []string{media}); err == nil {
		t.Fatal("expected error for empty file")
	}

	outside := filepath.Join(root, "other.mp4")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateLocalPath(outside, []string{media}); err == nil {
		t.Fatal("expected error for path outside roots")
	}
}
