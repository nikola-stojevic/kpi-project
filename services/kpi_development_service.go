package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"kpiproject/models"
	repository "kpiproject/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

type KPIService interface {
	CreateKPI(ctx context.Context, kpi *models.KPIDevelopment) (*models.KPIDevelopment, error)
	GetKPIByID(ctx context.Context, id primitive.ObjectID) (*models.KPIDevelopment, error)
	GetAllKPIs(ctx context.Context) ([]models.KPIDevelopment, error)
	UpdateKPI(ctx context.Context, id primitive.ObjectID, kpi *models.KPIDevelopment) (*models.KPIDevelopment, error)
	SoftDeleteKPI(ctx context.Context, id primitive.ObjectID, updatedBy string) error
	// File attachment methods
	UploadAttachment(ctx context.Context, kpiID primitive.ObjectID, filename string, fileData io.Reader, updatedBy string, contentType string) (*models.Attachment, error)
	DownloadAttachment(ctx context.Context, fileID primitive.ObjectID) (*gridfs.DownloadStream, error)
	DeleteAttachment(ctx context.Context, kpiID, fileID primitive.ObjectID, updatedBy string) error
	TransferAttachmentBetweenKPIs(ctx context.Context, fromKPIID, toKPIID, fileID primitive.ObjectID, updatedBy string) error
	// Analytics methods
	GetKPIPerformanceStats(ctx context.Context) ([]bson.M, error)
}

type kpiService struct {
	repo repository.KPIRepository
}

func NewKPIService(repo repository.KPIRepository) KPIService {
	return &kpiService{
		repo: repo,
	}
}

func (s *kpiService) CreateKPI(ctx context.Context, kpi *models.KPIDevelopment) (*models.KPIDevelopment, error) {
	now := time.Now()
	kpi.Metadata.CreatedAt = now
	kpi.Metadata.UpdatedAt = now
	kpi.IsDeleted = false

	// Initialize attachments as empty array if not already set
	if kpi.Attachments == nil {
		kpi.Attachments = []models.Attachment{}
	}

	err := s.repo.Create(ctx, kpi)
	if err != nil {
		return nil, err
	}

	return kpi, nil
}

func (s *kpiService) GetKPIByID(ctx context.Context, id primitive.ObjectID) (*models.KPIDevelopment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *kpiService) GetAllKPIs(ctx context.Context) ([]models.KPIDevelopment, error) {
	return s.repo.GetAll(ctx)
}

func (s *kpiService) UpdateKPI(ctx context.Context, id primitive.ObjectID, kpi *models.KPIDevelopment) (*models.KPIDevelopment, error) {
	existingKPI, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if kpi.Goal != "" {
		existingKPI.Goal = kpi.Goal
	}
	if kpi.Description != "" {
		existingKPI.Description = kpi.Description
	}
	if !kpi.DueDate.IsZero() {
		existingKPI.DueDate = kpi.DueDate
	}
	existingKPI.ActualPercent = kpi.ActualPercent
	existingKPI.Metadata.UpdatedBy = kpi.Metadata.UpdatedBy
	existingKPI.Metadata.UpdatedAt = time.Now()

	err = s.repo.Update(ctx, id, existingKPI)
	if err != nil {
		return nil, err
	}

	return existingKPI, nil
}

func (s *kpiService) SoftDeleteKPI(ctx context.Context, id primitive.ObjectID, updatedBy string) error {
	return s.repo.SoftDelete(ctx, id, updatedBy)
}

func (s *kpiService) UploadAttachment(ctx context.Context, kpiID primitive.ObjectID, filename string, fileData io.Reader, updatedBy string, contentType string) (*models.Attachment, error) {
	fmt.Printf("Starting file upload for KPI ID: %s\n", kpiID.Hex())

	// First: Verify that the KPI exists
	_, err := s.repo.GetByID(ctx, kpiID)
	if err != nil {
		fmt.Printf("KPI not found: %v\n", err)
		return nil, fmt.Errorf("KPI not found: %v", err)
	}
	fmt.Println("KPI exists, proceeding with file upload")

	// Second: Upload file to GridFS
	fileID, err := s.repo.UploadFile(ctx, filename, fileData, updatedBy, contentType)
	if err != nil {
		fmt.Printf("Failed to upload file: %v\n", err)
		return nil, fmt.Errorf("failed to upload file: %v", err)
	}
	fmt.Printf("File uploaded to GridFS with ID: %s\n", fileID.Hex())

	// Create attachment record
	attachment := models.Attachment{
		FileID:   fileID,
		Filename: filename,
	}

	// Third: Add attachment to KPI document
	err = s.repo.AddAttachment(ctx, kpiID, attachment, updatedBy)
	if err != nil {
		fmt.Printf("Failed to add attachment to KPI: %v\n", err)

		// CLEANUP: Delete the uploaded file since adding attachment failed
		fmt.Printf("Cleaning up uploaded file due to attachment failure...\n")
		if cleanupErr := s.repo.DeleteFile(context.Background(), fileID); cleanupErr != nil {
			fmt.Printf("Failed to cleanup uploaded file %s: %v\n", fileID.Hex(), cleanupErr)
		} else {
			fmt.Printf("Successfully cleaned up uploaded file %s\n", fileID.Hex())
		}

		return nil, fmt.Errorf("failed to add attachment to KPI: %v", err)
	}
	fmt.Println("Attachment added to KPI document")

	fmt.Printf("File upload completed successfully")
	return &attachment, nil
}

func (s *kpiService) DownloadAttachment(ctx context.Context, fileID primitive.ObjectID) (*gridfs.DownloadStream, error) {
	return s.repo.DownloadFile(ctx, fileID)
}

func (s *kpiService) DeleteAttachment(ctx context.Context, kpiID, fileID primitive.ObjectID, updatedBy string) error {
	fmt.Printf("Starting attachment deletion\n")
	fmt.Printf("KPI ID: %s\n", kpiID.Hex())
	fmt.Printf("File ID: %s\n", fileID.Hex())

	// First: Verify that the KPI exists and has the attachment
	kpi, err := s.repo.GetByID(ctx, kpiID)
	if err != nil {
		fmt.Printf("KPI not found: %v\n", err)
		return fmt.Errorf("KPI not found: %v", err)
	}
	fmt.Printf("KPI found: %s\n", kpi.Goal)

	// Check if the attachment exists in this KPI
	var attachmentExists bool
	var attachmentFilename string
	for _, attachment := range kpi.Attachments {
		if attachment.FileID == fileID {
			attachmentExists = true
			attachmentFilename = attachment.Filename
			break
		}
	}

	if !attachmentExists {
		fmt.Printf("Attachment not found in KPI\n")
		return fmt.Errorf("attachment with file_id %s not found in KPI %s", fileID.Hex(), kpiID.Hex())
	}
	fmt.Printf("Attachment found: %s\n", attachmentFilename)

	// Second: Remove attachment from KPI document first
	err = s.repo.RemoveAttachment(ctx, kpiID, fileID, updatedBy)
	if err != nil {
		fmt.Printf("Failed to remove attachment from KPI: %v\n", err)
		return fmt.Errorf("failed to remove attachment from KPI: %v", err)
	}
	fmt.Println("Attachment removed from KPI document")

	// Third: Delete file from GridFS
	err = s.repo.DeleteFile(ctx, fileID)
	if err != nil {
		fmt.Printf("Failed to delete file from GridFS: %v\n", err)

		// ROLLBACK: Re-add attachment to KPI since file deletion failed
		fmt.Printf("Rolling back: Re-adding attachment to KPI due to file deletion failure...\n")
		attachment := models.Attachment{
			FileID:   fileID,
			Filename: attachmentFilename,
		}
		if rollbackErr := s.repo.AddAttachment(ctx, kpiID, attachment, updatedBy); rollbackErr != nil {
			fmt.Printf("Failed to rollback attachment addition: %v\n", rollbackErr)
			return fmt.Errorf("failed to delete file from GridFS and rollback failed: %v (original error: %v)", rollbackErr, err)
		}
		fmt.Println("Successfully rolled back attachment to KPI")

		return fmt.Errorf("failed to delete file from GridFS: %v", err)
	}
	fmt.Println("File deleted from GridFS")

	fmt.Printf("Attachment deletion completed successfully\n")
	fmt.Printf("File '%s' deleted from KPI '%s'\n", attachmentFilename, kpi.Goal)

	return nil
}

func (s *kpiService) GetKPIPerformanceStats(ctx context.Context) ([]bson.M, error) {
	return s.repo.GetKPIPerformanceStats(ctx)
}

func (s *kpiService) TransferAttachmentBetweenKPIs(ctx context.Context, fromKPIID, toKPIID, fileID primitive.ObjectID, updatedBy string) error {
	// Create transaction context with timeout
	transactionCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the client for transaction support
	client := s.repo.GetClient()

	// Start a session for transaction
	session, err := client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(transactionCtx)

	// Create session context
	sessionCtx := mongo.NewSessionContext(transactionCtx, session)

	fmt.Printf("Starting attachment transfer transaction\n")
	fmt.Printf("From KPI: %s\n", fromKPIID.Hex())
	fmt.Printf("To KPI: %s\n", toKPIID.Hex())
	fmt.Printf("File ID: %s\n", fileID.Hex())

	// Start transaction
	if err := session.StartTransaction(); err != nil {
		fmt.Printf("Failed to start transaction: %v\n", err)
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	fmt.Println("Transaction started successfully")

	// Step 1: Verify both KPIs exist
	fromKPI, err := s.repo.GetByID(sessionCtx, fromKPIID)
	if err != nil {
		fmt.Printf("Source KPI not found: %v\n", err)
		session.AbortTransaction(sessionCtx)
		return fmt.Errorf("source KPI not found: %v", err)
	}
	fmt.Printf("Source KPI found: %s\n", fromKPI.Goal)

	toKPI, err := s.repo.GetByID(sessionCtx, toKPIID)
	if err != nil {
		fmt.Printf("Destination KPI not found: %v\n", err)
		session.AbortTransaction(sessionCtx)
		return fmt.Errorf("destination KPI not found: %v", err)
	}
	fmt.Printf("Destination KPI found: %s\n", toKPI.Goal)

	// Step 2: Find the attachment in the source KPI
	var attachmentToTransfer *models.Attachment
	for _, attachment := range fromKPI.Attachments {
		if attachment.FileID == fileID {
			attachmentToTransfer = &attachment
			break
		}
	}

	if attachmentToTransfer == nil {
		fmt.Printf("Attachment not found in source KPI\n")
		session.AbortTransaction(sessionCtx)
		return fmt.Errorf("attachment with file_id %s not found in source KPI", fileID.Hex())
	}
	fmt.Printf("Attachment found: %s\n", attachmentToTransfer.Filename)

	// Step 3: Remove attachment from source KPI
	err = s.repo.RemoveAttachment(sessionCtx, fromKPIID, fileID, updatedBy)
	if err != nil {
		fmt.Printf("Failed to remove attachment from source KPI: %v\n", err)
		session.AbortTransaction(sessionCtx)
		return fmt.Errorf("failed to remove attachment from source KPI: %v", err)
	}
	fmt.Println("Attachment removed from source KPI")

	// Step 4: Add attachment to destination KPI
	err = s.repo.AddAttachment(sessionCtx, toKPIID, *attachmentToTransfer, updatedBy)
	if err != nil {
		fmt.Printf("Failed to add attachment to destination KPI: %v\n", err)
		session.AbortTransaction(sessionCtx)
		return fmt.Errorf("failed to add attachment to destination KPI: %v", err)
	}
	fmt.Println("Attachment added to destination KPI")

	// Step 5: Commit transaction
	if err := session.CommitTransaction(sessionCtx); err != nil {
		fmt.Printf("Failed to commit transaction: %v\n", err)
		session.AbortTransaction(sessionCtx)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("Attachment transfer completed successfully\n")
	fmt.Printf("File '%s' moved from '%s' to '%s'\n",
		attachmentToTransfer.Filename, fromKPI.Goal, toKPI.Goal)

	return nil
}
