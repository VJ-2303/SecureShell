package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// CopyDir recursively copies the directory tree from src to dst.
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src failed: %w", err)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("mkdirall dst failed: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("readdir src failed: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CopyFile copies a single file from src to dst, preserving mode.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src file failed: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat src file failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdirall parent failed: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("create dst file failed: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("io copy failed: %w", err)
	}

	return nil
}

// ClearDir removes all files and subdirectories inside dir without deleting dir itself.
func ClearDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		p := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}

	return nil
}

// ScanDir scans a directory and returns sorted lists of relative directory and file paths.
func ScanDir(dir string) (dirs []string, files []string, err error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil, nil
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		relSlash := filepath.ToSlash(rel)
		if info.IsDir() {
			dirs = append(dirs, relSlash)
		} else {
			files = append(files, relSlash)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	sort.Strings(dirs)
	sort.Strings(files)
	return dirs, files, nil
}

// CompareDirs compares all files and directories in dirA and dirB.
// Returns true if the structures and file contents are identical.
func CompareDirs(dirA, dirB string) (bool, string, error) {
	dirsA, filesA, err := ScanDir(dirA)
	if err != nil {
		return false, "", err
	}
	dirsB, filesB, err := ScanDir(dirB)
	if err != nil {
		return false, "", err
	}

	// Compare directory lists
	if len(dirsA) != len(dirsB) {
		return false, fmt.Sprintf("directory count mismatch: %d vs %d", len(dirsA), len(dirsB)), nil
	}
	for i := range dirsA {
		if dirsA[i] != dirsB[i] {
			return false, fmt.Sprintf("directory mismatch: %s vs %s", dirsA[i], dirsB[i]), nil
		}
	}

	// Compare file lists
	if len(filesA) != len(filesB) {
		return false, fmt.Sprintf("file count mismatch: %d vs %d", len(filesA), len(filesB)), nil
	}
	for i := range filesA {
		if filesA[i] != filesB[i] {
			return false, fmt.Sprintf("file mismatch: %s vs %s", filesA[i], filesB[i]), nil
		}

		// Compare content
		contentA, err := os.ReadFile(filepath.Join(dirA, filesA[i]))
		if err != nil {
			return false, "", err
		}
		contentB, err := os.ReadFile(filepath.Join(dirB, filesB[i]))
		if err != nil {
			return false, "", err
		}
		if !bytes.Equal(contentA, contentB) {
			return false, fmt.Sprintf("content mismatch in file %s", filesA[i]), nil
		}
	}

	return true, "", nil
}
