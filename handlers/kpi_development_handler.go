package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	middleware "kpiproject/middlewares"
	"kpiproject/models"
	service "kpiproject/services"
	"kpiproject/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KPIHandler struct {
	service service.KPIService
}

func NewKPIHandler(service service.KPIService) *KPIHandler {
	return &KPIHandler{
		service: service,
	}
}

func (h *KPIHandler) CreateKPI(w http.ResponseWriter, r *http.Request) {
	var kpi models.KPIDevelopment
	if err := utils.DecodeAndValidate(w, r, &kpi); err != nil {
		return
	}

	// Get username from JWT context
	username := middleware.GetUsernameFromContext(r.Context())
	kpi.Metadata.CreatedBy = username
	kpi.Metadata.UpdatedBy = username

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	createdKPI, err := h.service.CreateKPI(ctx, &kpi)
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.HandleDataResponse(w, "KPI created successfully", createdKPI, http.StatusCreated)
}

func (h *KPIHandler) GetKPIByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid KPI ID format", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	kpi, err := h.service.GetKPIByID(ctx, objectID)
	if err != nil {
		utils.HandleMessageResponse(w, "KPI not found", http.StatusNotFound)
		return
	}

	utils.HandleDataResponse(w, "KPI retrieved successfully", kpi, http.StatusOK)
}

func (h *KPIHandler) GetAllKPIs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	kpis, err := h.service.GetAllKPIs(ctx)
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.HandleDataResponse(w, "KPIs retrieved successfully", kpis, http.StatusOK)
}

func (h *KPIHandler) UpdateKPI(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid KPI ID format", http.StatusBadRequest)
		return
	}

	var kpi models.KPIDevelopment
	if err := utils.DecodeAndValidate(w, r, &kpi); err != nil {
		return
	}

	// Get username from JWT context
	username := middleware.GetUsernameFromContext(r.Context())
	kpi.Metadata.UpdatedBy = username

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	updatedKPI, err := h.service.UpdateKPI(ctx, objectID, &kpi)
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.HandleDataResponse(w, "KPI updated successfully", updatedKPI, http.StatusOK)
}

func (h *KPIHandler) DeleteKPI(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid KPI ID format", http.StatusBadRequest)
		return
	}

	// Get username from JWT context
	username := middleware.GetUsernameFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err = h.service.SoftDeleteKPI(ctx, objectID, username) // Pass username
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.HandleMessageResponse(w, "KPI deleted successfully", http.StatusOK)
}

func (h *KPIHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		utils.HandleMessageResponse(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get KPI ID from URL
	id := r.PathValue("id")
	kpiID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid KPI ID format", http.StatusBadRequest)
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.HandleMessageResponse(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file size (optional)
	if header.Size > 10<<20 { // 10 MB
		utils.HandleMessageResponse(w, "File size too large (max 10MB)", http.StatusBadRequest)
		return
	}

	// Get username from JWT context
	username := middleware.GetUsernameFromContext(r.Context())

	// Get content type from header
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Default content type
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Upload the file with metadata
	attachment, err := h.service.UploadAttachment(ctx, kpiID, header.Filename, file, username, contentType)
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.HandleDataResponse(w, "File uploaded successfully", attachment, http.StatusOK)
}

func (h *KPIHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	// Get file ID from URL
	fileIDStr := r.PathValue("fileId")
	fileID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid file ID format", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Download the file
	downloadStream, err := h.service.DownloadAttachment(ctx, fileID)
	if err != nil {
		utils.HandleMessageResponse(w, "File not found", http.StatusNotFound)
		return
	}
	defer downloadStream.Close()

	// Get file info
	fileInfo := downloadStream.GetFile()

	// Get content type from metadata, default to application/octet-stream
	contentType := "application/octet-stream"
	if fileInfo.Metadata != nil && len(fileInfo.Metadata) > 0 {
		var metaMap map[string]interface{}
		if err := bson.Unmarshal(fileInfo.Metadata, &metaMap); err == nil {
			if ctRaw, exists := metaMap["contentType"]; exists {
				if contentTypeStr, ok := ctRaw.(string); ok && contentTypeStr != "" {
					contentType = contentTypeStr
				}
			}
		}
	}

	// Set response headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Name))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Length, 10))

	// Copy file data to response
	_, err = io.Copy(w, downloadStream)
	if err != nil {
		utils.HandleMessageResponse(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
}

func (h *KPIHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	// Get KPI ID from URL
	kpiIDStr := r.PathValue("id")
	kpiID, err := primitive.ObjectIDFromHex(kpiIDStr)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid KPI ID format", http.StatusBadRequest)
		return
	}

	// Get file ID from URL
	fileIDStr := r.PathValue("fileId")
	fileID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid file ID format", http.StatusBadRequest)
		return
	}

	// Get username from JWT context
	username := middleware.GetUsernameFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Delete the attachment
	err = h.service.DeleteAttachment(ctx, kpiID, fileID, username)
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.HandleMessageResponse(w, "Attachment deleted successfully", http.StatusOK)
}

func (h *KPIHandler) GetKPIPerformanceStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	stats, err := h.service.GetKPIPerformanceStats(ctx)
	if err != nil {
		utils.HandleMessageResponse(w, fmt.Sprintf("Failed to get KPI performance stats: %v", err), http.StatusInternalServerError)
		return
	}

	utils.HandleDataResponse(w, "KPI performance statistics retrieved successfully", stats, http.StatusOK)
}

func (h *KPIHandler) TransferAttachment(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var transferRequest struct {
		FromKPIID string `json:"from_kpi_id" validate:"required"`
		ToKPIID   string `json:"to_kpi_id" validate:"required"`
		FileID    string `json:"file_id" validate:"required"`
	}

	if err := utils.DecodeAndValidate(w, r, &transferRequest); err != nil {
		return
	}

	// Convert string IDs to ObjectIDs
	fromKPIID, err := primitive.ObjectIDFromHex(transferRequest.FromKPIID)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid from_kpi_id format", http.StatusBadRequest)
		return
	}

	toKPIID, err := primitive.ObjectIDFromHex(transferRequest.ToKPIID)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid to_kpi_id format", http.StatusBadRequest)
		return
	}

	fileID, err := primitive.ObjectIDFromHex(transferRequest.FileID)
	if err != nil {
		utils.HandleMessageResponse(w, "Invalid file_id format", http.StatusBadRequest)
		return
	}

	// Validate that source and destination are different
	if fromKPIID == toKPIID {
		utils.HandleMessageResponse(w, "Source and destination KPI cannot be the same", http.StatusBadRequest)
		return
	}

	// Get username from JWT context
	username := middleware.GetUsernameFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Transfer the attachment
	err = h.service.TransferAttachmentBetweenKPIs(ctx, fromKPIID, toKPIID, fileID, username)
	if err != nil {
		utils.HandleMessageResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseData := map[string]interface{}{
		"from_kpi_id":    fromKPIID.Hex(),
		"to_kpi_id":      toKPIID.Hex(),
		"file_id":        fileID.Hex(),
		"transferred_at": time.Now(),
	}

	utils.HandleDataResponse(w, "Attachment transferred successfully", responseData, http.StatusOK)
}
