package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"os"
)

var collection *mongo.Collection

type Document map[string]interface{}

func main() {
	initMongoDB()

	r := gin.Default()
	r.Use(CORSMiddleware())

	r.GET("/", test)
	r.POST("/spv", createDocument)
	r.PUT("/spv/:id", updateDocument)
	r.GET("/spv", getAllDocuments)
	r.GET("/spv/:id", getDocumentByID)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Panicf("error: %s", err)
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func initMongoDB() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI("mongodb+srv://cherifim:aEiogaT5aQXq6sev@welledge.a9nzvcy.mongodb.net/?retryWrites=true&w=majority&appName=welledge").SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	//defer func() {
	//	if err = client.Disconnect(context.TODO()); err != nil {
	//		panic(err)
	//	}
	//}()

	collection = client.Database("omni").Collection("spv1")

	// Send a ping to confirm a successful connection
	if err := collection.Database().RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	fmt.Println("Collection assigned:", collection != nil)
	fmt.Println("Using collection:", collection.Name())
}

func test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello Mito!"})
}

func createDocument(c *gin.Context) {
	var doc Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// You must ensure there's an `_id` or other unique key to match on
	id, ok := doc["_id"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing _id field for upsert"})
		return
	}

	filter := bson.M{"_id": id}
	opts := options.Replace().SetUpsert(true)

	result, err := collection.ReplaceOne(context.Background(), filter, doc, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upsert failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func createDocument_without_replacing(c *gin.Context) {
	var doc Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := collection.InsertOne(context.Background(), doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Insertion failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func updateDocument(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var doc map[string]interface{}
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delete(doc, "_id") // prevent _id from being updated
	update := bson.M{"$set": doc}
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func getAllDocuments(c *gin.Context) {
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch"})
		return
	}
	defer cursor.Close(context.Background())

	var documents []Document
	if err = cursor.All(context.Background(), &documents); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Decoding failed"})
		return
	}
	c.JSON(http.StatusOK, documents)
}

func getDocumentByID(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var doc Document
	err = collection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	c.JSON(http.StatusOK, doc)
}
