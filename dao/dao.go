package dao

import (
	"context"
	"log"

	"github.com/ambalabanov/scanner/models"
	"go.mongodb.org/mongo-driver/bson"
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

//Delete documents
func Delete(f bson.M) (int64, error) {
	log.Println("Delete documents")
	res, err := collection.DeleteMany(context.TODO(), f)
	return res.DeletedCount, err
}

//Find documents
func Find(f bson.M) ([]models.Document, error) {
	log.Println("Read from database")
	var doc []models.Document
	cursor, err := collection.Find(context.TODO(), f)
	err = cursor.All(context.TODO(), &doc)
	return doc, err
}
