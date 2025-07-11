package internals

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Mode = string

const (
	VIDEO Mode = "video"
	IMAGE Mode = "image"
)

var (
	videoExts = []string{
		".mp4", ".mov", ".avi", ".mkv",
		".webm", ".flv", ".wmv",
	}

	imageExts = []string{
		".jpg", ".jpeg", ".png", ".gif",
		".bmp", ".webp", ".tiff",
	}
)

func CheckIfElementExist(slice []string, element string) bool {
	sort.Strings(slice)
	idx := sort.SearchStrings(slice, element)
	return idx < len(slice) && slice[idx] == element
}

func isValidPathFormat(path string) (bool, error) {
	if strings.ContainsAny(path, `<>:"|?*`) {
		return false, errors.New("ERROR: special characters like (<>:'|?*) are not allowed")
	}

	// Must be valid filepath format
	if !filepath.IsLocal(path) {
		return false, errors.New("ERROR: path is invalid")
	}

	// Must have extension
	ext := filepath.Ext(path)
	if ext == "" {
		return false, errors.New("ERROR: file at the end of path needs to have an extension")
	}
	return true, nil
}

func checkFileIsOfTypeMode(path string, mode Mode) bool {
	ext := filepath.Ext(path)

	modeOptions := make([]string, 0)
	switch mode {
	case VIDEO:
		modeOptions = videoExts
	case IMAGE:
		modeOptions = imageExts
	}

	return CheckIfElementExist(modeOptions, ext)
}

func videoPathCheck(path string) error {
	path = strings.TrimSpace(path)

	_, err := isValidPathFormat(path)
	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)
	path = filepath.Clean(path)

	if !checkFileIsOfTypeMode(path, VIDEO) {
		return errors.New("ERROR: file extension needs to be a video")
	}

	return nil
}

func imagePathCheck(path string) error {
	path = strings.TrimSpace(path)

	_, err := isValidPathFormat(path)
	if err != nil {
		return err
	}

	osPath, _ := os.Getwd()
	path = filepath.Join(osPath, path)
	path = filepath.Clean(path)

	if !checkFileIsOfTypeMode(path, IMAGE) {
		return errors.New("ERROR: file extension needs to be a image")
	}

	return nil
}
