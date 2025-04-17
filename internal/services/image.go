package services

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/chai2010/webp"
	"github.com/kartex/imageprovider/internal/models"
	"github.com/kartex/imageprovider/internal/storage"
)

const (
	defaultMaxCacheSize = 100
	defaultMaxCacheMB   = 100 // 100MB default size limit
)

type ImageService struct {
	images     []*models.Image
	mu         sync.RWMutex
	primary    storage.Storage
	secondary  storage.Storage
	maxSize    int
	maxBytes   int64
	totalBytes int64
}

func NewImageService(primary storage.Storage, secondary storage.Storage) *ImageService {
	maxCacheSize := defaultMaxCacheSize
	if maxCacheStr := os.Getenv("MAX_CACHE_FILES"); maxCacheStr != "" {
		if size, err := strconv.Atoi(maxCacheStr); err == nil && size > 0 {
			maxCacheSize = size
		}
	}

	maxCacheMB := defaultMaxCacheMB
	if maxMBStr := os.Getenv("MAX_CACHE_SIZE_MB"); maxMBStr != "" {
		if mb, err := strconv.Atoi(maxMBStr); err == nil && mb > 0 {
			maxCacheMB = mb
		}
	}

	return &ImageService{
		images:     make([]*models.Image, 0, maxCacheSize),
		primary:    primary,
		secondary:  secondary,
		maxSize:    maxCacheSize,
		maxBytes:   int64(maxCacheMB) * 1024 * 1024, // Convert MB to bytes
		totalBytes: 0,
	}
}

func (s *ImageService) AddImage(image *models.Image) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if image already exists
	for i, img := range s.images {
		if img.ID == image.ID {
			// Update existing image
			s.totalBytes -= int64(len(img.Data))
			s.images[i] = image
			s.totalBytes += int64(len(image.Data))
			return nil
		}
	}

	// Check if adding this image would exceed size limit
	imageSize := int64(len(image.Data))
	if s.totalBytes+imageSize > s.maxBytes {
		// Remove oldest images until we have enough space
		for len(s.images) > 0 && s.totalBytes+imageSize > s.maxBytes {
			oldest := s.images[len(s.images)-1]
			s.totalBytes -= int64(len(oldest.Data))
			s.images = s.images[:len(s.images)-1]
		}
		// If still not enough space, return error
		if s.totalBytes+imageSize > s.maxBytes {
			return fmt.Errorf("image too large for cache (size: %d bytes, available: %d bytes)",
				imageSize, s.maxBytes-s.totalBytes)
		}
	}

	// Add to the beginning of the slice
	s.images = append([]*models.Image{image}, s.images...)
	s.totalBytes += imageSize

	// If we've exceeded the file count limit, remove oldest images
	if len(s.images) > s.maxSize {
		for len(s.images) > s.maxSize {
			oldest := s.images[len(s.images)-1]
			s.totalBytes -= int64(len(oldest.Data))
			s.images = s.images[:len(s.images)-1]
		}
	}

	return nil
}

func (s *ImageService) GetImage(id string) (*models.Image, error) {
	// Get base filename without extension
	baseID := strings.TrimSuffix(id, filepath.Ext(id))

	s.mu.RLock()
	// Check cache for any version of this file (with any extension)
	for _, img := range s.images {
		imgBaseID := strings.TrimSuffix(img.ID, filepath.Ext(img.ID))
		if imgBaseID == baseID {
			s.mu.RUnlock()
			// If it's an image and not in WebP format, convert it
			if _, _, err := image.Decode(bytes.NewReader(img.Data)); err == nil {
				if img.Format == "webp" {
					// If it's already in WebP format, return it directly
					return img, nil
				}
				// Convert to WebP if not already
				decoded, _, err := image.Decode(bytes.NewReader(img.Data))
				if err != nil {
					log.Printf("Warning: Failed to decode cached image: %v", err)
					return nil, err
				}

				buf := new(bytes.Buffer)
				if err := webp.Encode(buf, decoded, &webp.Options{Lossless: true}); err != nil {
					log.Printf("Warning: Failed to encode cached image to WebP: %v", err)
					return nil, err
				}

				img.Data = buf.Bytes()
				img.Format = "webp"
			}
			return img, nil
		}
	}
	s.mu.RUnlock()

	// If not in cache, try primary storage
	img, err := s.primary.Get(id)
	if err == nil {
		// If it's an image, ensure it's in WebP format
		if _, _, err := image.Decode(bytes.NewReader(img.Data)); err == nil {
			if img.Format != "webp" {
				decoded, _, err := image.Decode(bytes.NewReader(img.Data))
				if err != nil {
					log.Printf("Warning: Failed to decode image from primary storage: %v", err)
					return nil, err
				}

				buf := new(bytes.Buffer)
				if err := webp.Encode(buf, decoded, &webp.Options{Lossless: true}); err != nil {
					log.Printf("Warning: Failed to encode image to WebP: %v", err)
					return nil, err
				}

				img.Data = buf.Bytes()
				img.Format = "webp"
			}
		}

		// Add to cache
		s.mu.Lock()
		if len(s.images) >= s.maxSize {
			s.images = s.images[1:]
		}
		s.images = append(s.images, img)
		s.mu.Unlock()
		return img, nil
	}

	// If not in primary storage and secondary storage is available, try it
	if s.secondary != nil {
		log.Printf("Image not found in primary storage, trying secondary storage for ID: %s", id)
		img, err = s.secondary.Get(id)
		if err == nil {
			// If it's an image, ensure it's in WebP format
			if _, _, err := image.Decode(bytes.NewReader(img.Data)); err == nil {
				if img.Format != "webp" {
					decoded, _, err := image.Decode(bytes.NewReader(img.Data))
					if err != nil {
						log.Printf("Warning: Failed to decode image from secondary storage: %v", err)
						return nil, err
					}

					buf := new(bytes.Buffer)
					if err := webp.Encode(buf, decoded, &webp.Options{Lossless: true}); err != nil {
						log.Printf("Warning: Failed to encode image to WebP: %v", err)
						return nil, err
					}

					img.Data = buf.Bytes()
					img.Format = "webp"
				}
			} else {
				// Not an image file, keep original format
				log.Printf("File is not an image, keeping original format: %s", img.Format)
			}

			// Add to cache
			s.mu.Lock()
			if len(s.images) >= s.maxSize {
				s.images = s.images[1:]
			}
			s.images = append(s.images, img)
			s.mu.Unlock()

			// Save to primary storage for future access
			if err := s.primary.Save(img); err != nil {
				log.Printf("Warning: Failed to save to primary storage after retrieving from secondary: %v", err)
			}
			return img, nil
		}
		log.Printf("Image not found in secondary storage for ID: %s", id)
	}

	return nil, err
}

func (s *ImageService) DeleteImage(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from cache
	for i, img := range s.images {
		if img.ID == id {
			s.totalBytes -= int64(len(img.Data))
			s.images = append(s.images[:i], s.images[i+1:]...)
			break
		}
	}

	// Delete from primary storage
	if err := s.primary.Delete(id); err != nil {
		return err
	}

	// Delete from secondary storage if available
	if s.secondary != nil {
		if err := s.secondary.Delete(id); err != nil {
			log.Printf("Warning: Failed to delete image from secondary storage: %v", err)
		}
	}

	return nil
}

func (s *ImageService) GetImages() []*models.Image {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.images
}
