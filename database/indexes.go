package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateKPIIndexes(db *mongo.Database) error {
	collection := db.Collection("kpi_developments")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		// ANALYTICS: actual_percent + is_deleted
		// Used by: GetKPIPerformanceStats aggregation pipeline
		{
			Keys: bson.D{
				{Key: "is_deleted", Value: 1},
				{Key: "actual_percent", Value: 1},
			},
			Options: options.Index().SetName("idx_is_deleted_actual_percent"),
		},

		// ANALYTICS: due_date + is_deleted
		// Used by: GetKPIPerformanceStats aggregation pipeline
		{
			Keys: bson.D{
				{Key: "is_deleted", Value: 1},
				{Key: "due_date", Value: 1},
			},
			Options: options.Index().SetName("idx_is_deleted_due_date"),
		},

		// ATTACHMENT OPERATIONS: file_id lookups
		// Used by: File validation, attachment operations
		{
			Keys: bson.D{
				{Key: "attachments.file_id", Value: 1},
				{Key: "is_deleted", Value: 1},
			},
			Options: options.Index().SetName("idx_attachments_file_id_is_deleted"),
		},

		// UPDATE OPERATIONS: _id + is_deleted combination
		// Used by: SoftDelete, AddAttachment, RemoveAttachment
		{
			Keys: bson.D{
				{Key: "_id", Value: 1},
				{Key: "is_deleted", Value: 1},
			},
			Options: options.Index().SetName("idx_id_is_deleted"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create KPI indexes: %v", err)
	}

	fmt.Println("KPI indexes created successfully")
	return nil
}
