package repository

import (
	"context"
	"fmt"
	"io"
	"time"

	"kpiproject/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type KPIRepository interface {
	Create(ctx context.Context, kpi *models.KPIDevelopment) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.KPIDevelopment, error)
	GetAll(ctx context.Context) ([]models.KPIDevelopment, error)
	Update(ctx context.Context, id primitive.ObjectID, kpi *models.KPIDevelopment) error
	SoftDelete(ctx context.Context, id primitive.ObjectID, updatedBy string) error
	GetClient() *mongo.Client
	// GridFS methods
	UploadFile(ctx context.Context, filename string, fileData io.Reader, uploadedBy string, contentType string) (primitive.ObjectID, error)
	DownloadFile(ctx context.Context, fileID primitive.ObjectID) (*gridfs.DownloadStream, error)
	DeleteFile(ctx context.Context, fileID primitive.ObjectID) error
	// Attachment methods
	AddAttachment(ctx context.Context, kpiID primitive.ObjectID, attachment models.Attachment, updatedBy string) error
	RemoveAttachment(ctx context.Context, kpiID primitive.ObjectID, fileID primitive.ObjectID, updatedBy string) error
	// Analytics methods
	GetKPIPerformanceStats(ctx context.Context) ([]bson.M, error)
}

type kpiRepository struct {
	collection *mongo.Collection
	bucket     *gridfs.Bucket
}

func NewKPIRepository(db *mongo.Database) KPIRepository {
	// Create GridFS bucket
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		panic(fmt.Sprintf("Failed to create GridFS bucket: %v", err))
	}

	return &kpiRepository{
		collection: db.Collection("kpi_developments"),
		bucket:     bucket,
	}
}

func (r *kpiRepository) Create(ctx context.Context, kpi *models.KPIDevelopment) error {
	kpi.ID = primitive.NewObjectID()

	_, err := r.collection.InsertOne(ctx, kpi)
	return err
}

func (r *kpiRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.KPIDevelopment, error) {

	var kpi models.KPIDevelopment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&kpi)
	if err != nil {
		return nil, err
	}

	return &kpi, nil
}

func (r *kpiRepository) GetAll(ctx context.Context) ([]models.KPIDevelopment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var kpis []models.KPIDevelopment
	if err = cursor.All(ctx, &kpis); err != nil {
		return nil, err
	}

	return kpis, nil
}

func (r *kpiRepository) Update(ctx context.Context, id primitive.ObjectID, kpi *models.KPIDevelopment) error {

	filter := bson.M{"_id": id}
	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": kpi})
	if err != nil {
		return err
	}

	// Check if any document was actually updated
	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id %s", id.Hex())
	}

	return nil
}

func (r *kpiRepository) SoftDelete(ctx context.Context, id primitive.ObjectID, updatedBy string) error {
	update := bson.M{
		"$set": bson.M{
			"is_deleted":          true,
			"metadata.updated_at": time.Now(),
			"metadata.updated_by": updatedBy, // Add this field
		},
	}

	filter := bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// Check if document was found and updated
	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id %s or already deleted", id.Hex())
	}

	return nil
}

func (r *kpiRepository) GetClient() *mongo.Client {
	return r.collection.Database().Client()
}

// GridFS methods
func (r *kpiRepository) UploadFile(ctx context.Context, filename string, fileData io.Reader, uploadedBy string, contentType string) (primitive.ObjectID, error) {
	uploadOpts := options.GridFSUpload().SetMetadata(bson.M{
		"uploadedBy":  uploadedBy,
		"uploadedAt":  time.Now(),
		"contentType": contentType,
	})

	fileID, err := r.bucket.UploadFromStream(filename, fileData, uploadOpts)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to upload file to GridFS: %v", err)
	}

	return fileID, nil
}

func (r *kpiRepository) DownloadFile(ctx context.Context, fileID primitive.ObjectID) (*gridfs.DownloadStream, error) {
	downloadStream, err := r.bucket.OpenDownloadStream(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from GridFS: %v", err)
	}

	return downloadStream, nil
}

func (r *kpiRepository) DeleteFile(ctx context.Context, fileID primitive.ObjectID) error {
	err := r.bucket.Delete(fileID)
	if err != nil {
		return err
	}

	return nil
}

func (r *kpiRepository) AddAttachment(ctx context.Context, kpiID primitive.ObjectID, attachment models.Attachment, updatedBy string) error {
	filter := bson.M{"_id": kpiID, "is_deleted": bson.M{"$ne": true}}
	update := bson.M{
		"$push": bson.M{
			"attachments": attachment,
		},
		"$set": bson.M{
			"metadata.updated_at": time.Now(),
			"metadata.updated_by": updatedBy,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id %s", kpiID.Hex())
	}

	return nil
}

func (r *kpiRepository) RemoveAttachment(ctx context.Context, kpiID primitive.ObjectID, fileID primitive.ObjectID, updatedBy string) error {
	filter := bson.M{"_id": kpiID, "is_deleted": bson.M{"$ne": true}}
	update := bson.M{
		"$pull": bson.M{
			"attachments": bson.M{"file_id": fileID},
		},
		"$set": bson.M{
			"metadata.updated_at": time.Now(),
			"metadata.updated_by": updatedBy,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id %s", kpiID.Hex())
	}

	return nil
}

// Get KPI statistics grouped by completion status
func (r *kpiRepository) GetKPIPerformanceStats(ctx context.Context) ([]bson.M, error) {
	pipeline := mongo.Pipeline{
		// Match non-deleted KPIs
		bson.D{{Key: "$match", Value: bson.M{"is_deleted": bson.M{"$ne": true}}}},

		// Add computed fields
		bson.D{{Key: "$addFields", Value: bson.M{
			"status": bson.M{
				"$switch": bson.M{
					"branches": []bson.M{
						{"case": bson.M{"$gte": []interface{}{"$actual_percent", 100}}, "then": "Completed"},
						{"case": bson.M{"$gte": []interface{}{"$actual_percent", 50}}, "then": "On Track"},
						{"case": bson.M{"$gte": []interface{}{"$actual_percent", 25}}, "then": "At Risk"},
						{"case": bson.M{"$gte": []interface{}{"$actual_percent", 1}}, "then": "Behind"},
						{"case": bson.M{"$eq": []interface{}{"$actual_percent", 0}}, "then": "Not Started"},
					},
					"default": "Not Started",
				},
			},
			"days_until_due": bson.M{
				"$divide": []interface{}{
					bson.M{"$subtract": []interface{}{"$due_date", "$$NOW"}},
					1000 * 60 * 60 * 24, // Convert milliseconds to days
				},
			},
			"attachments_count": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$isArray": "$attachments"},
					"then": bson.M{"$size": "$attachments"},
					"else": 0,
				},
			},
		}}},

		// Group by status
		bson.D{{Key: "$group", Value: bson.M{
			"_id":                "$status",
			"count":              bson.M{"$sum": 1},
			"avg_completion":     bson.M{"$avg": "$actual_percent"},
			"total_attachments":  bson.M{"$sum": "$attachments_count"},
			"avg_days_until_due": bson.M{"$avg": "$days_until_due"},
		}}},

		// Sort by count descending
		bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
