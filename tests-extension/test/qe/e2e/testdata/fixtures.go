package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	fixtureDir  string
	fixtureMu   sync.Mutex
	fixtureInit bool
)

func ensureFixtureDir() string {
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	if fixtureInit {
		return fixtureDir
	}
	var err error
	fixtureDir, err = os.MkdirTemp("", "testdata-fixtures-")
	if err != nil {
		panic(fmt.Sprintf("failed to create fixture directory: %v", err))
	}
	if err := os.Chmod(fixtureDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to set fixture directory permissions: %v", err))
	}
	fixtureInit = true
	return fixtureDir
}

func FixturePath(elem ...string) string {
	dir := ensureFixtureDir()
	relativePath := filepath.Join(elem...)
	if filepath.IsAbs(relativePath) || strings.Contains(relativePath, "..") {
		panic(fmt.Sprintf("invalid fixture path: %s", relativePath))
	}
	targetPath := filepath.Join(dir, relativePath)

	if _, err := os.Stat(targetPath); err == nil {
		return targetPath
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		panic(fmt.Sprintf("failed to create directory for %s: %v", relativePath, err))
	}

	bindataPath := relativePath
	tempDir, err := os.MkdirTemp("", "bindata-extract-")
	if err != nil {
		panic(fmt.Sprintf("failed to create temp directory: %v", err))
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean up temp dir %s: %v\n", tempDir, err)
		}
	}()

	if err := RestoreAsset(tempDir, bindataPath); err != nil {
		if err := RestoreAssets(tempDir, bindataPath); err != nil {
			panic(fmt.Sprintf("failed to restore fixture %s: %v", relativePath, err))
		}
	}

	extractedPath := filepath.Join(tempDir, bindataPath)

	if err := filepath.Walk(extractedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0755)
		}
		return os.Chmod(path, 0644)
	}); err != nil {
		panic(fmt.Sprintf("failed to set permissions on extracted files: %v", err))
	}

	if err := os.Rename(extractedPath, targetPath); err != nil {
		panic(fmt.Sprintf("failed to move extracted files: %v", err))
	}

	if info, err := os.Stat(targetPath); err == nil {
		var chmodErr error
		if info.IsDir() {
			chmodErr = os.Chmod(targetPath, 0755)
		} else {
			chmodErr = os.Chmod(targetPath, 0644)
		}
		if chmodErr != nil {
			panic(fmt.Sprintf("failed to set final permissions on %s: %v", targetPath, chmodErr))
		}
	}

	return targetPath
}

func CleanupFixtures() error {
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	if fixtureDir != "" {
		err := os.RemoveAll(fixtureDir)
		fixtureDir = ""
		fixtureInit = false
		return err
	}
	return nil
}

func GetFixtureData(elem ...string) ([]byte, error) {
	relativePath := filepath.Join(elem...)
	cleanPath := relativePath
	if len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}
	return Asset(cleanPath)
}

func MustGetFixtureData(elem ...string) []byte {
	data, err := GetFixtureData(elem...)
	if err != nil {
		panic(fmt.Sprintf("failed to get fixture data: %v", err))
	}
	return data
}

func FixtureExists(elem ...string) bool {
	relativePath := filepath.Join(elem...)
	cleanPath := relativePath
	if len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}
	_, err := Asset(cleanPath)
	return err == nil
}

func ListFixtures() []string {
	names := AssetNames()
	fixtures := make([]string, 0, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, "testdata/") {
			fixtures = append(fixtures, strings.TrimPrefix(name, "testdata/"))
		}
	}
	sort.Strings(fixtures)
	return fixtures
}
