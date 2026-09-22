package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/templates"
)

// FileState represents the current on-disk state of a single document file.
type FileState struct {
	RelPath string // relative to .archcore/, e.g. "auth/jwt-strategy.adr.md"
	AbsPath string
	Hash    string
}

// HashFile computes the SHA-256 hex digest of the file at the given path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file for hash: %w", err)
	}
	defer func() { _ = f.Close() }() // read-only handle

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ScanFiles walks .archcore/ recursively and returns the current state of
// every .md file found. Skips hidden directories, settings.json,
// .sync-state.json, the reserved global/ tree, and any document under a
// declared global source — globals are read-only mounts owned by another
// repository and must never be pushed to the server as local documents.
func ScanFiles(baseDir string) ([]FileState, error) {
	archcoreDir := filepath.Join(baseDir, ".archcore")
	var files []FileState

	// A guard, not an advisory: this list is the only thing keeping a read-only
	// mount out of the push, so an unreadable settings.json must stop the scan
	// rather than report that no globals are declared.
	globals, err := config.LoadGlobals(baseDir)
	if err != nil {
		return nil, fmt.Errorf("cannot verify global sources: %w", err)
	}
	var globalDirs []string
	for _, gs := range globals {
		globalDirs = append(globalDirs, filepath.ToSlash(config.ResolveGlobalPath(baseDir, gs.Path)))
	}

	err = templates.WalkArchcoreFilesSkipping(archcoreDir, []string{"global"}, func(path string, d fs.DirEntry) error {
		slashPath := filepath.ToSlash(path)
		for _, dir := range globalDirs {
			if slashPath == dir || strings.HasPrefix(slashPath, dir+"/") {
				return nil
			}
		}

		relPath, _ := filepath.Rel(archcoreDir, path)
		relPath = filepath.ToSlash(relPath)

		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", relPath, err)
		}

		files = append(files, FileState{
			RelPath: relPath,
			AbsPath: path,
			Hash:    hash,
		})
		return nil
	})

	return files, err
}
