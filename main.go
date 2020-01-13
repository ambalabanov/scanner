package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ambalabanov/go-nmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	wg         sync.WaitGroup
	config     configuration
	nmapXML    nmap.NmapRun
	collection *mongo.Collection
	usenmap    bool
)

type configuration struct {
	Db struct {
		URI  string `json:"uri"`
		Db   string `json:"db"`
		Coll string `json:"coll"`
	} `json:"database"`
	Hosts []struct {
		Name  string `json:"name"`
		Ports []int  `json:"ports"`
	} `json:"hosts"`
}
type document struct {
	ID      primitive.ObjectID `bson:"_id"`
	Host    string             `bson:"host"`
	Port    int                `bson:"port"`
	Status  string             `bson:"status"`
	Server  string             `bson:"server"`
	Content string             `bson:"content"`
	URL     string             `bson:"url"`
}

func init() {
	var err error
	fmt.Print("Load 'config.json' file...")
	config, err = loadJSON("config.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if _, err := os.Stat("nmap_output.xml"); os.IsNotExist(err) {
		usenmap = true
		fmt.Println("File 'nmap_output.xml' not found!")
		fmt.Println("Use hosts from 'config.json' file")
		usenmap = false
	} else {
		usenmap = true
		fmt.Print("Load 'nmap_output.xml' file...")
		nmapXML, err = loadXML("nmap_output.xml")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK!")
	}
	fmt.Print("Connect to mongodb...")
	err = dbConnect()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Drop collection...")
	err = dbDrop()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	log.Println("Start scan...")
	if usenmap {
		for _, h := range nmapXML.Hosts {
			for _, p := range h.Ports {
				go checkHTTP(string(h.Hostnames[0].Name), int(p.PortId))
			}
		}
	} else {
		for _, h := range config.Hosts {
			for _, p := range h.Ports {
				go checkHTTP(string(h.Name), int(p))
			}
		}
	}
	fmt.Printf("Active gorutines %v\n", runtime.NumGoroutine())
	time.Sleep(1 * time.Second)
	wg.Wait()
	log.Println("Scan complete!")
	fmt.Print("Retrive data from database...")
	filter := bson.M{"status": bson.M{"$ne": ""}}
	result, err := dbFind(filter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Println("______________")
	fmt.Println("Print results:")
	fmt.Println("______________")
	for _, r := range result {
		fmt.Println(r.Host, r.Port, r.Server)
	}
	for _, r := range result {
		fmt.Println(r.URL, r.Status, r.Content)
	}
}

func loadJSON(filename string) (configuration, error) {
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

func loadXML(filename string) (nmap.NmapRun, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nmap.NmapRun{}, err
	}
	var x nmap.NmapRun
	err = xml.Unmarshal(bytes, &x)
	if err != nil {
		return nmap.NmapRun{}, err
	}
	return x, nil
}

func checkHTTP(host string, port int) {
	wg.Add(1)
	defer wg.Done()
	url := fmt.Sprintf("http://%s:%d", host, port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Head(url)
	if err != nil {
		return
	}
	go dbInsert(bson.M{"host": host, "port": port, "url": url, "status": r.Status, "server": r.Header.Get("Server"), "content": r.Header.Get("Content-Type")})
}

func dbInsert(data bson.M) error {
	wg.Add(1)
	defer wg.Done()
	_, err := collection.InsertOne(context.TODO(), data)
	if err != nil {
		return err
	}
	return nil
}

func dbDelete(filter bson.M) error {
	_, err := collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		return err
	}
	return nil
}

func dbDrop() error {
	err := collection.Drop(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func dbFind(filter bson.M) ([]*document, error) {
	opts := options.Find()
	opts.SetShowRecordID(false)
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		return []*document{}, err
	}
	var results []*document
	for cursor.Next(context.TODO()) {
		var result document
		if err := cursor.Decode(&result); err != nil {
			return []*document{}, err
		}
		results = append(results, &result)
	}
	return results, nil
}

func dbConnect() error {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(config.Db.URI))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return err
	}
	collection = client.Database(config.Db.Db).Collection(config.Db.Coll)
	return nil
}
