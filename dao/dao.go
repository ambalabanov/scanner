package dao

import (
	"context"

	"github.com/ambalabanov/scanner/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

//Connect to db
func Connect(URI string, Db string, Coll string) (*mongo.Collection, error) {
	client, _ := mongo.Connect(context.TODO(), options.Client().ApplyURI(URI))
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, err
	}
	collection := client.Database(Db).Collection(Coll)
	return collection, nil
}

// Drop collection
func Drop(c *mongo.Collection) error {
	if err := c.Drop(context.TODO()); err != nil {
		return err
	}
	return nil
}

//Write one document
func Write(c *mongo.Collection, d models.Document) error {
	data, err := bson.Marshal(d)
	if err != nil {
		return err
	}
	_, err = c.InsertOne(context.TODO(), data)
	if err != nil {
		return err
	}
	return nil
}
