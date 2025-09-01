package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"kpiproject/database"
	"kpiproject/handlers"
	repository "kpiproject/repositories"
	routes "kpiproject/routes"
	services "kpiproject/services"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Get MongoDB credentials from environment variables
	username := os.Getenv("MONGO_USERNAME")
	password := os.Getenv("MONGO_PASSWORD")
	cluster := os.Getenv("MONGO_CLUSTER")
	appName := os.Getenv("MONGO_APP_NAME")
	jwtSecret := os.Getenv("JWT_SECRET")

	if username == "" || password == "" || cluster == "" || appName == "" {
		log.Fatal("Missing required environment variables")
	}

	// Build MongoDB Atlas connection string
	uri := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=%s",
		username, password, cluster, appName)

	// Create a new client and connect to the server
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Fatal("Failed to disconnect from MongoDB:", err)
		}
	}()

	// Set a timeout for the ping operation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ping the primary to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	fmt.Println("Successfully connected to MongoDB Atlas!")

	// Check replica set status
	checkIfReplicaSet(client)

	// Initialize database
	db := client.Database("kpi_project")

	// Create indexes
	fmt.Println("Creating database indexes...")
	if err := database.CreateKPIIndexes(db); err != nil {
		log.Printf("Warning: Failed to create KPI indexes: %v", err)
	}

	// Initialize repository, service, and handler
	kpiRepo := repository.NewKPIRepository(db)
	kpiService := services.NewKPIService(kpiRepo)
	kpiHandler := handlers.NewKPIHandler(kpiService)

	// Setup routes using ServeMux with JWT middleware
	mux := routes.SetupKPIRoutes(kpiHandler, jwtSecret)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func checkIfReplicaSet(client *mongo.Client) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result bson.M
	// Use the newer "hello" command instead of deprecated "isMaster"
	err := client.Database("admin").RunCommand(ctx, bson.M{"hello": 1}).Decode(&result)

	if err != nil {
		fmt.Printf("Error checking replica set: %v\n", err)
		return false
	}

	// Check if this is a replica set
	if setName, exists := result["setName"]; exists {
		fmt.Printf("Part of replica set: %v\n", setName)
		return true
	}

	fmt.Println("Not part of a replica set")
	return false
}
