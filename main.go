package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net"
	"sync"
)

const (
	uri        = "mongodb://localhost:27017"
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
	ports := []int{22, 80, 443, 8080}
	host := "scanme.nmap.org"
	for _, p := range ports {
		wg.Add(1)
		go scan(host, p)
	}
	wg.Wait()
}

func dbInsert(ep Endpoint) {
	client := dbConnect()
	collection := client.Database(Db).Collection(Collection)
	insertResult, err := collection.InsertOne(context.TODO(), ep)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("dbInsert: ", insertResult.InsertedID, ep)
}

func scan(host string, port int) {
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	conn.Close()
	result := Endpoint{host, port}
	dbInsert(result)
}

func dbConnect() *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err)
	}
	return client
}
