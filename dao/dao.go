package dao

import (
	"context"
	"log"

	"github.com/ambalabanov/scanner/models"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var collection *mongo.Collection

//Connect to db
func Connect(URI string, Db string, Coll string) error {
	log.Println("Connect to mongodb")
	client, _ := mongo.Connect(context.TODO(), options.Client().ApplyURI(URI))
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return err
	}
	collection = client.Database(Db).Collection(Coll)
	return nil
}

// Drop collection
func Drop() error {
	log.Println("Drop collection")
	if err := collection.Drop(context.TODO()); err != nil {
		return err
	}
	return nil
}

//InsertOne document
func InsertOne(d interface{}) error {
	log.Println("Write to database")
	_, err := collection.InsertOne(context.TODO(), d)
	if err != nil {
		return err
	}
	return nil
}

//InsertMany documents
func InsertMany(d []interface{}) error {
	log.Println("Write to database")
	_, err := collection.InsertMany(context.TODO(), d)
	if err != nil {
		return err
	}
	return nil
}

//Delete documents
func Delete(f interface{}) (int64, error) {
	log.Println("Delete documents")
	res, err := collection.DeleteMany(context.TODO(), f)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

//Find documents
func Find(f interface{}) ([]models.Document, error) {
	log.Println("Read from database")
	var doc []models.Document
	cursor, err := collection.Find(context.TODO(), f)
	if err != nil {
		return doc, err
	}
	if err = cursor.All(context.TODO(), &doc); err != nil {
		return doc, err
	}
	return doc, nil
}
