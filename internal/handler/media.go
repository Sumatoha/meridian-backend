package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/service"
	"github.com/meridian/api/internal/storage"
)

const (
	maxImageSize = 20 << 20  // 20 MB
	maxVideoSize = 100 << 20 // 100 MB
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

var allowedVideoTypes = map[string]bool{
	"video/mp4":  true,
	"video/mov":  true,
	"video/mpeg": true,
}

type MediaHandler struct {
	slotSvc *service.SlotService
	storage *storage.Client
	queries *repository.Queries
	logger  *slog.Logger
}

func NewMediaHandler(slotSvc *service.SlotService, storageClient *storage.Client, queries *repository.Queries, logger *slog.Logger) *MediaHandler {
	return &MediaHandler{slotSvc: slotSvc, storage: storageClient, queries: queries, logger: logger}
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	slotID, ok := parseUUID(chi.URLParam(r, "slot_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid slot ID")
		return
	}

	// Limit total body size to max video size
	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize)

	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		respondError(w, http.StatusBadRequest, "file_too_large", "file exceeds maximum size")
		return
	}

	// Get existing slot to append media
	slot, err := h.queries.GetSlotByID(r.Context(), slotID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "slot not found")
		return
	}

	var existingMedia []dto.MediaItem
	json.Unmarshal(slot.Media, &existingMedia)

	userID := auth.UserID(r.Context())
	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {
		contentType := fileHeader.Header.Get("Content-Type")
		isImage := allowedImageTypes[contentType]
		isVideo := allowedVideoTypes[contentType]

		if !isImage && !isVideo {
			respondError(w, http.StatusBadRequest, "invalid_type", fmt.Sprintf("unsupported file type: %s", contentType))
			return
		}

		if isImage && fileHeader.Size > maxImageSize {
			respondError(w, http.StatusBadRequest, "file_too_large", "image exceeds 20MB limit")
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", "failed to read file")
			return
		}
		defer file.Close()

		storagePath, err := h.storage.Upload(r.Context(), userID, fileHeader.Filename, contentType, file)
		if err != nil {
			h.logger.Error("upload failed", slog.String("error", err.Error()))
			respondError(w, http.StatusInternalServerError, "upload_error", "failed to upload file")
			return
		}

		mediaType := "image"
		if isVideo {
			mediaType = "video"
		}

		existingMedia = append(existingMedia, dto.MediaItem{
			StoragePath:      storagePath,
			Type:             mediaType,
			Order:            len(existingMedia),
			OriginalFilename: fileHeader.Filename,
			URL:              h.storage.GetPublicURL(storagePath),
		})
	}

	result, err := h.slotSvc.UpdateSlotMedia(r.Context(), slotID, existingMedia)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update slot media")
		return
	}

	respondJSON(w, http.StatusOK, dto.MediaUploadResponse{Media: result.Media})
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slotID, ok := parseUUID(chi.URLParam(r, "slot_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid slot ID")
		return
	}

	indexStr := chi.URLParam(r, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_index", "invalid media index")
		return
	}

	slot, err := h.queries.GetSlotByID(r.Context(), slotID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "slot not found")
		return
	}

	var media []dto.MediaItem
	json.Unmarshal(slot.Media, &media)

	if index < 0 || index >= len(media) {
		respondError(w, http.StatusBadRequest, "invalid_index", "media index out of range")
		return
	}

	// Delete from storage
	h.storage.Delete(r.Context(), media[index].StoragePath)

	// Remove from slice
	media = append(media[:index], media[index+1:]...)

	// Reorder
	for i := range media {
		media[i].Order = i
	}

	result, err := h.slotSvc.UpdateSlotMedia(r.Context(), slotID, media)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update slot media")
		return
	}

	respondJSON(w, http.StatusOK, dto.MediaUploadResponse{Media: result.Media})
}

func isAllowedMediaExt(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
		".mp4": true, ".mov": true,
	}
	return allowed[ext]
}
