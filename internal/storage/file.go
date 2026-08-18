package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirPerm  = 0755
	filePerm = 0644
)

func OpenFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return file, nil
}

func CreateFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return nil, fmt.Errorf("create directory for %s: %w", path, err)
	}

	file, err := os.OpenFile(path,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		filePerm)

	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}

	return file, nil
}

func OpenAppendFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		filePerm,
	)
	if err != nil {
		return nil, fmt.Errorf("open append %s: %w", path, err)
	}

	return file, nil
}

func SyncFile(file *os.File) error {
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", file.Name(), err)
	}

	return nil
}

func SyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory %s: %w", path, err)
	}
	defer dir.Close()

	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync directory %s: %w", path, err)
	}

	return nil
}

func CloseFile(file *os.File) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", file.Name(), err)
	}

	return nil
}

func RemoveFile(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, data, filePerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func AtomicWrite(path string, write func(*os.File) error) error {

	dir := filepath.Dir(path)

	if err := os.MkdirAll(path, dirPerm); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmp.Name()

	cleanup := func() {
		_ = CloseFile(tmp)
		_ = RemoveFile(tmpPath)
	}

	if err := write(tmp); err != nil {
		cleanup()
		return fmt.Errorf("write %s: %w", path, err)
	}

	if err := SyncFile(tmp); err != nil {
		cleanup()
		return err
	}

	if err := CloseFile(tmp); err != nil {
		_ = RemoveFile(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = RemoveFile(tmpPath)
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}

	if err := SyncDir(dir); err != nil {
		return err
	}

	return nil
}
