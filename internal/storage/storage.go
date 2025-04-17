package storage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kartex/imageprovider/internal/models"
)

type Storage interface {
	Save(image *models.Image) error
	Get(id string) (*models.Image, error)
	Delete(id string) error
	List() ([]string, error)
}

type FileSystemStorage struct {
	baseDir string
}

func NewFileSystemStorage(baseDir string) (*FileSystemStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &FileSystemStorage{baseDir: baseDir}, nil
}

func (s *FileSystemStorage) getPath(id string) string {
	// Split ID into parts for directory structure
	// e.g., "123456" -> "12/34/56.webp"
	if len(id) < 2 {
		return filepath.Join(s.baseDir, id+".webp")
	}

	parts := make([]string, 0)
	for i := 0; i < len(id); i += 2 {
		end := i + 2
		if end > len(id) {
			end = len(id)
		}
		parts = append(parts, id[i:end])
	}

	// Last part is the filename
	filename := parts[len(parts)-1] + ".webp"
	parts = parts[:len(parts)-1]

	// Create the full path
	return filepath.Join(append([]string{s.baseDir}, append(parts, filename)...)...)
}

func (s *FileSystemStorage) Save(image *models.Image) error {
	path := s.getPath(image.ID)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, image.Data, 0644)
}

func (s *FileSystemStorage) Get(id string) (*models.Image, error) {
	path := s.getPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &models.Image{
		ID:     id,
		Data:   data,
		Format: "webp",
	}, nil
}

func (s *FileSystemStorage) Delete(id string) error {
	path := s.getPath(id)
	return os.Remove(path)
}

func (s *FileSystemStorage) List() ([]string, error) {
	var ids []string
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".webp") {
			// Convert path back to ID
			relPath, err := filepath.Rel(s.baseDir, path)
			if err != nil {
				return err
			}
			// Remove .webp extension and directory separators
			id := strings.ReplaceAll(strings.TrimSuffix(relPath, ".webp"), string(filepath.Separator), "")
			ids = append(ids, id)
		}
		return nil
	})

	return ids, err
}
