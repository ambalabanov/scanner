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
	log.Println("Write from database")
	_, err := collection.InsertOne(context.TODO(), d)
	if err != nil {
		return err
	}
	return nil
}

//InsertMany documents
func InsertMany(d []interface{}) error {
	log.Println("Write from database")
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
func Find(f interface{}) ([]interface{}, error) {
	log.Println("Read from database")
	doc := make([]interface{}, 0)
	cursor, err := collection.Find(context.TODO(), f)
	if err != nil {
		return doc, err
	}
	for cursor.Next(context.TODO()) {
		var result models.Document
		if err := cursor.Decode(&result); err != nil {
			return doc, err
		}
		doc = append(doc, result)
	}
	return doc, nil
}
