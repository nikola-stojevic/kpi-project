# KPI Development API

A RESTful API for tracking Key Performance Indicators (KPIs) built with Go and MongoDB. This project serves as a proof-of-concept similar to company platforms like Workday, designed for personal development tracking.

## Introduction

This KPI Development API allows users to:
- Set and track personal development goals
- Monitor completion percentages for each KPI
- Upload supporting documents (certificates, reports, etc.)
- Transfer attachments between KPIs using MongoDB transactions
- Generate analytics and performance statistics

The project demonstrates key MongoDB concepts including CRUD operations, soft deletes, GridFS file storage, transactions with replica sets, database indexing, and complex aggregation pipelines.

## Technologies Used

- **Go 1.21+** - Backend language
- **MongoDB Atlas** - Cloud database with default replica set support
- **GridFS** - File storage system
- **JWT** - Authentication middleware
- **OpenAPI 3.0** - API documentation

## Features

### Core Functionality
- **CRUD Operations** - Create, Read, Update, Delete KPIs
- **Soft Delete** - Mark records as deleted without permanent removal
- **File Management** - Upload, download, and delete attachments using GridFS
- **Transactions** - Atomic operations for attachment transfers between KPIs
- **Authentication** - JWT-based security for all endpoints
- **Analytics** - performance statistics with MongoDB aggregation pipelines
- **Database Indexing** - for query performance

## API Endpoints

### KPI Management

#### `POST /api/kpi`
**Create a new KPI**
- Creates a KPI development record with goal, description, and due date

#### `GET /api/kpi`
**Get all KPIs**
- Retrieves all non-deleted KPI records

#### `GET /api/kpi/{id}`
**Get KPI by ID**
- Fetches specific KPI using MongoDB ObjectID

#### `PUT /api/kpi/{id}`
**Update KPI**
- Updates existing KPI fields (goal, description, due_date, actual_percent)

#### `DELETE /api/kpi/{id}`
**Soft delete KPI**
- Sets `is_deleted: true` instead of permanent removal

---

### File Attachment Management

#### `POST /api/kpi/{id}/attachments`
**Upload file attachment**
- Uploads files to GridFS with metadata (uploadedBy, uploadedAt, contentType)
- Links attachment to specific KPI record
- Atomic operation with cleanup on failure

#### `GET /api/kpi/attachments/{fileId}/download`
**Download file attachment**
- Streams file directly from GridFS
- Sets appropriate content headers (Content-Type, Content-Disposition)
- Preserves original filename and MIME type
- Efficient for large file downloads

#### `DELETE /api/kpi/{id}/attachments/{fileId}`
**Delete file attachment**
- Removes attachment from both KPI record and GridFS
- Two-phase operation with rollback capability
- Maintains data consistency between document and file storage

---

### Advanced File Operations

#### `POST /api/kpi/attachments/transfer`
**Transfer attachment between KPIs**
- **MongoDB Transaction**: Ensures atomicity across multiple operations
- **Replica Set Required**: Uses MongoDB Atlas replica set for transaction support
- **Rollback Support**: Automatic rollback on any operation failure

**Transaction Steps:**
1. Verify both source and destination KPIs exist
2. Validate attachment exists in source KPI
3. Remove attachment from source KPI
4. Add attachment to destination KPI
5. Commit transaction or rollback on failure

---

### Analytics & Reporting

#### `GET /api/kpi/analytics/performance`
**Get KPI performance statistics**
- **Complex Aggregation Pipeline**: Demonstrates advanced MongoDB queries
- **Status Classification**: Groups KPIs by completion status
- **Statistical Analysis**: Calculates averages and totals

**Status Categories:**
- **Completed** (100% done)
- **On Track** (50-99% done)
- **At Risk** (25-49% done)
- **Behind** (1-24% done)
- **Not Started** (0% done)

**Aggregation Features:**
- Groups by computed status field
- Calculates average completion percentage
- Counts total attachments per status
- Computes average days until due date
- Sorts results by KPI count

**Sample Response:**
```json
{
  "status_code": 200,
  "message": "KPI performance statistics retrieved successfully",
  "data": [
    {
      "_id": "On Track",
      "count": 15,
      "avg_completion": 67.5,
      "total_attachments": 25,
      "avg_days_until_due": 45.2
    },
    {
      "_id": "Completed",
      "count": 8,
      "avg_completion": 100.0,
      "total_attachments": 12,
      "avg_days_until_due": -5.3
    }
  ]
}
```

---

## Database Design

### Collections
- **`kpi_developments`** - Main KPI records with embedded attachments
- **`fs.files`** - GridFS file metadata
- **`fs.chunks`** - GridFS file data chunks

### Key Indexes
1. **`{is_deleted: 1, actual_percent: 1}`** - Analytics queries
2. **`{is_deleted: 1, due_date: 1}`** - Date-based operations
3. **`{attachments.file_id: 1, is_deleted: 1}`** - File operations
4. **`{_id: 1, is_deleted: 1}`** - Update operations

## Authentication

All endpoints require JWT authentication via Authorization header:
```
Authorization: Bearer <jwt_token>
```

The JWT token should contain:
- `username` - Used for audit trails and file metadata

## Setup Instructions

### Prerequisites
- Go 1.21 or higher
- MongoDB Atlas account (replica set enabled by default)
- Environment variables configured

### Environment Variables
```env
MONGO_USERNAME=your_username
MONGO_PASSWORD=your_password
MONGO_CLUSTER=your_cluster
MONGO_APP_NAME=your_app_name
JWT_SECRET=your_jwt_secret
```

### Installation
```bash
# Clone the repository
git clone <repository_url>
cd kpi-project

# Install dependencies
go mod tidy

# Run the application
go run main.go
```

## Project Structure
```
kpi-project/
├── handlers/           # HTTP request handlers
├── services/          # Business logic layer
├── repositories/      # Data access layer
├── models/           # Data structures
├── middlewares/      # JWT authentication
├── routes/           # Route definitions
├── database/         # Index creation
├── utils/            # Utility functions
├── docs/             # API documentation
└── main.go           # Application entry point
```

## Learning Outcomes

This project demonstrates:

1. **MongoDB CRUD Operations**
   - Proper error handling and validation
   - Soft delete patterns for data retention

2. **GridFS File Storage**
   - Large file handling with metadata
   - Efficient streaming for downloads

3. **Transaction Management**
   - Multi-document ACID transactions
   - Replica set requirements and setup

4. **Database Optimization**
   - Strategic index placement
   - Query performance analysis

5. **Aggregation Pipelines**
   - Complex data transformations
   - Statistical calculations and grouping

## API Documentation

Complete OpenAPI 3.0 documentation is available in `docs/swagger.yaml`. The documentation includes:
- Detailed endpoint descriptions
- Request/response schemas
- Authentication requirements
- Example requests and responses

---
