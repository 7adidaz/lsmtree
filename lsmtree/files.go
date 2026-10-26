package lsmtree

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func DataPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, "data"), nil
}

const sstablePrefix = "sstable_"

func fileName(prefix string, id uint64) string {
	return fmt.Sprintf("%s%020d", prefix, id)
}

func parseID(name, prefix string) (uint64, bool) {
	id, found := strings.CutPrefix(name, prefix)
	if !found {
		return 0, false
	}
	num, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, false
	}

	return num, true
}

func nextFreeID(dataPath string) (uint64, error) {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return 0, err
	}

	var next uint64
	for _, e := range entries {
		if id, found := parseID(e.Name(), sstablePrefix); found && id >= next {
			next = id + 1
		}
	}

	return next, nil
}

func sstableFiles(dataPath string) ([]string, error) {
	return filesByID(dataPath, sstablePrefix)
}

func filesByID(dataPath, prefix string) ([]string, error) {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return nil, err
	}

	type fileEntry struct {
		name string
		id   uint64
	}

	var files []fileEntry
	for _, e := range entries {
		id, found := parseID(e.Name(), prefix)
		if !found {
			continue
		}
		files = append(files, fileEntry{name: e.Name(), id: id})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].id < files[j].id
	})

	paths := make([]string, 0, len(files))
	for _, e := range files {
		paths = append(paths, filepath.Join(dataPath, e.name))
	}

	return paths, nil
}
