package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d2hle/yt-automation/internal/domain"
)

func ValidateRef(ref domain.MediaRef, allowedRoots []string, required bool) (resolved string, err error) {
	if ref.S3 != nil {
		if ref.LocalPath != "" {
			return "", fmt.Errorf("media ref must not set both local_path and s3")
		}
		return "", fmt.Errorf("s3 media source is not implemented yet")
	}
	if ref.LocalPath == "" {
		if required {
			return "", fmt.Errorf("local_path is required")
		}
		return "", nil
	}
	return ValidateLocalPath(ref.LocalPath, allowedRoots)
}

func ValidateLocalPath(path string, allowedRoots []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	clean := filepath.Clean(path)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if !isUnderAllowedRoots(abs, allowedRoots) {
		return "", fmt.Errorf("path %q is outside allowed media roots", path)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("path not accessible: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory")
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("file is empty")
	}
	return abs, nil
}

func isUnderAllowedRoots(abs string, roots []string) bool {
	for _, root := range roots {
		rootAbs, err := filepath.Abs(filepath.Clean(root))
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(rootAbs, abs)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		return true
	}
	return false
}
