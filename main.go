package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	uri  = "mongodb://localhost:27017"
	db   = "scannerDb"
	coll = "endpoints"
)

var (
	wg sync.WaitGroup
)

type endpoint struct {
	Host string
	Port int
}

func main() {
	collection, err := dbConnect()
	if err != nil {
		log.Fatalln("Db not connected!")
	}
	ports := []int{22, 80, 443, 8080}
	hosts := []string{"scanme.nmap.org", "getinside.cloud"}
	for _, h := range hosts {
		collection.DeleteMany(context.TODO(), bson.M{"host": h})
		for _, p := range ports {
			go scan(collection, h, p)
		}
	}
	wg.Wait()
	filter := bson.M{}
	results := dbFind(collection, filter)
	for i := range results {
		fmt.Println(results[i].Host, results[i].Port)
	}

}

func scan(collection *mongo.Collection, host string, port int) {
	wg.Add(1)
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	conn.Close()
	result := endpoint{host, port}
	dbInsert(collection, result)
}

func dbInsert(collection *mongo.Collection, ep endpoint) {

	insertResult, err := collection.InsertOne(context.TODO(), ep)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("dbInsert: ", insertResult.InsertedID, ep)
}

func dbFind(collection *mongo.Collection, filter bson.M) []*endpoint {
	var results []*endpoint
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal("Error on Finding all the documents", err)
	}
	for cur.Next(context.TODO()) {
		var result endpoint
		err = cur.Decode(&result)
		if err != nil {
			log.Fatal("Error on Decoding the document", err)
		}
		results = append(results, &result)
	}
	return results
}

func dbConnect() (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return nil, err
	}
	collection := client.Database(db).Collection(coll)
	return collection, nil
}
