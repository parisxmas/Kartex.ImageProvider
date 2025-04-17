package handlers

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/kartex/imageprovider/internal/models"
	"github.com/kartex/imageprovider/internal/services"
)

type ImageHandler struct {
	imageService *services.ImageService
}

func NewImageHandler(imageService *services.ImageService) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
	}
}

func (h *ImageHandler) CreateImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Read the file content
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Decode the image (supports multiple formats)
	img, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format"})
		return
	}

	// Convert to WebP
	webpBuf := new(bytes.Buffer)
	if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: true}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to WebP"})
		return
	}

	// Create image model
	image := &models.Image{
		ID:     strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
		Data:   webpBuf.Bytes(),
		Format: format,
	}

	// Save to service
	if err := h.imageService.AddImage(image); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     image.ID,
		"format": format,
	})
}

func (h *ImageHandler) GetImage(c *gin.Context) {
	id := c.Param("id")
	image, err := h.imageService.GetImage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Set WebP content type
	c.Header("Content-Type", "image/webp")
	c.Data(http.StatusOK, "image/webp", image.Data)
}

func (h *ImageHandler) DeleteImage(c *gin.Context) {
	id := c.Param("id")
	if err := h.imageService.DeleteImage(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ImageHandler) ListImages(c *gin.Context) {
	images := h.imageService.GetImages()
	imageIDs := make([]string, len(images))
	for i, img := range images {
		imageIDs[i] = img.ID
	}

	c.JSON(http.StatusOK, gin.H{
		"images": imageIDs,
	})
}
