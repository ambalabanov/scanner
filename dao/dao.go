package dao

import (
	"context"
	"log"

	"github.com/ambalabanov/scanner/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection

//Connect to db
func Connect(URI string, Db string, Coll string) error {
	log.Println("Connect to mongodb")
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(URI))
	collection = client.Database(Db).Collection(Coll)
	return err
}

// Drop collection
func Drop() error {
	log.Println("Drop collection")
	err := collection.Drop(context.TODO())
	return err
}

//InsertOne document
func InsertOne(d models.Document) error {
	log.Println("Write to database")
	_, err := collection.InsertOne(context.TODO(), d)
	return err
}

//DeleteOne document
func DeleteOne(id string) (int64, error) {
	log.Println("Delete documents")
	docID, _ := primitive.ObjectIDFromHex(id)
	res, err := collection.DeleteMany(context.TODO(), bson.M{"_id": docID})
	return res.DeletedCount, err
}

//DeleteAll documents
func DeleteAll() (int64, error) {
	log.Println("Delete documents")
	res, err := collection.DeleteOne(context.TODO(), bson.M{})
	return res.DeletedCount, err
}

//FindAll documents
func FindAll() ([]models.Document, error) {
	log.Println("Read from database")
	var doc []models.Document
	cursor, err := collection.Find(context.TODO(), bson.M{})
	err = cursor.All(context.TODO(), &doc)
	return doc, err
}

//FindOne document
func FindOne(id string) ([]models.Document, error) {
	log.Println("Read from database")
	var doc []models.Document
	docID, _ := primitive.ObjectIDFromHex(id)
	cursor, err := collection.Find(context.TODO(), bson.M{"_id": docID})
	err = cursor.All(context.TODO(), &doc)
	return doc, err
}
