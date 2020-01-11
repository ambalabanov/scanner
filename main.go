package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"runtime"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	wg     sync.WaitGroup
	config configuration
)

type database struct {
	URI  string `json:"uri"`
	DB   string `json:"db"`
	Coll string `json:"coll"`
}
type endpoints struct {
	Host  string `json:"host"`
	Ports []int  `json:"ports"`
}
type configuration struct {
	DB        database    `json:"database"`
	Endpoints []endpoints `json:"endpoints"`
}

func init() {
	config, _ = loadConfig("config.json")
}

func main() {
	log.Println("Prepare db...")
	dbDelete(bson.M{})
	for _, h := range config.Endpoints {
		for _, p := range h.Ports {
			go checkTCP(string(h.Host), int(p))
		}
	}
	log.Printf("Start scan: active gorutines %v\n", runtime.NumGoroutine())
	wg.Wait()
	log.Println("Retrive data from db...")
	dbFind(bson.M{})
	log.Println("Done!")
}

func loadConfig(filename string) (configuration, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return configuration{}, err
	}
	var c configuration
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return configuration{}, err
	}
	return c, nil
}

func checkTCP(host string, port int) {
	wg.Add(1)
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err == nil {
		conn.Close()
		dbInsert(bson.M{"host": host, "port": port})
		go checkHTTP(host, port)
	}
}

func checkHTTP(host string, port int) {
	wg.Add(1)
	defer wg.Done()
	url := fmt.Sprintf("http://%s:%d", host, port)
	r, err := http.Head(url)
	if err != nil {
		return
	}
	if r.StatusCode != http.StatusNotFound {
		dbInsert(bson.M{"host": host, "port": port, "url": url, "status": r.Status, "header": r.Header})
	}
}

func dbInsert(data bson.M) {
	collection, err := dbConnect()
	if err != nil {
		log.Fatalln("Db not connected!")
	}
	collection.InsertOne(context.TODO(), data)
	if err != nil {
		log.Fatal(err)
	}
}
func dbDelete(filter bson.M) {
	collection, err := dbConnect()
	if err != nil {
		log.Fatalln("Db not connected!")
	}
	collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
	}
}

func dbFind(filter bson.M) {
	collection, err := dbConnect()
	if err != nil {
		log.Fatalln("db not connected!")
	}
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal("Error on Finding all the documents", err)
	}
	for cur.Next(context.TODO()) {
		var result bson.M
		err = cur.Decode(&result)
		if err != nil {
			log.Fatal("Error on Decoding the document", err)
		}
		fmt.Println(result)
	}
}

func dbConnect() (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(config.DB.URI))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return nil, err
	}
	collection := client.Database(config.DB.DB).Collection(config.DB.Coll)
	return collection, nil
}
