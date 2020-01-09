package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net"
	"sync"
	"time"
)

const (
	URI        = "mongodb://localhost:27017"
	Db         = "scannerDb"
	Collection = "endpoints"
)

var (
	wg sync.WaitGroup
)

type Endpoint struct {
	Host string
	Port int
}

func main() {
	client, err := dbConnect()
	if err != nil {
		log.Fatalln("Db not connected!")
	}
	ports := []int{22, 80, 443, 8080}
	host := "scanme.nmap.org"
	for _, p := range ports {
		go scan(client, host, p)
	}
	wg.Wait()
	filter := bson.M{"host": host}
	results := dbFind(client, filter)
	for i := range results {
		fmt.Println(results[i].Host, results[i].Port)
	}

}

func scan(client *mongo.Client, host string, port int) {
	wg.Add(1)
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	conn.Close()
	result := Endpoint{host, port}
	dbInsert(client, result)
}

func dbInsert(client *mongo.Client, ep Endpoint) {
	collection := client.Database(Db).Collection(Collection)
	insertResult, err := collection.InsertOne(context.TODO(), ep)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("dbInsert: ", insertResult.InsertedID, ep)
}

func dbFind(client *mongo.Client, filter bson.M) []*Endpoint {
	var results []*Endpoint
	collection := client.Database(Db).Collection(Collection)
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal("Error on Finding all the documents", err)
	}
	for cur.Next(context.TODO()) {
		var result Endpoint
		err = cur.Decode(&result)
		if err != nil {
			log.Fatal("Error on Decoding the document", err)
		}
		results = append(results, &result)
	}
	return results
}

func dbConnect() (*mongo.Client, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(URI))
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}
	return client, nil
}
