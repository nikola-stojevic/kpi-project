package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KPIDevelopment struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Goal          string             `json:"goal" bson:"goal" validate:"required"`
	Description   string             `json:"description" bson:"description" validate:"required"`
	DueDate       time.Time          `json:"due_date" bson:"due_date" validate:"required"`
	ActualPercent int                `json:"actual_percent" bson:"actual_percent" validate:"min=0,max=100"`
	Attachments   []Attachment       `json:"attachments" bson:"attachments"`
	IsDeleted     bool               `json:"is_deleted" bson:"is_deleted"`
	Metadata      Metadata           `json:"metadata" bson:"metadata"`
}

type Metadata struct {
	CreatedBy string    `json:"created_by" bson:"created_by"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type Attachment struct {
	FileID   primitive.ObjectID `bson:"file_id" json:"file_id"`   // GridFS file ID
	Filename string             `bson:"filename" json:"filename"` // Original filename
}
