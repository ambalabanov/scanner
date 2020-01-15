package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/ambalabanov/go-nmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	wg         sync.WaitGroup
	collection *mongo.Collection
	hosts      []host
)

type configuration struct {
	Db    database `json:"database"`
	Hosts []host   `json:"hosts"`
	Nmap  struct {
		Use  bool   `json:"use"`
		File string `json:"file"`
	} `json:"nmap"`
}
type host struct {
	Name  string `json:"name"`
	Ports []int  `json:"ports"`
}
type database struct {
	URI  string `json:"uri"`
	Db   string `json:"db"`
	Coll string `json:"coll"`
}
type document struct {
	Name   string      `bson:"name"`
	Port   int         `bson:"port"`
	URL    string      `bson:"url"`
	Method string      `bson:"method"`
	Scheme string      `bson:"scheme"`
	Host   string      `bson:"host"`
	Status int         `bson:"status"`
	Header http.Header `bson:"header"`
	Body   []byte      `bson:"body"`
}

type documents []document

func init() {
	fmt.Print("Load config.json...")
	var config configuration
	if err := config.Load("config.json"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if config.Nmap.Use {
		fmt.Printf("Use hosts from %s\n", config.Nmap.File)
		bytes, err := ioutil.ReadFile(config.Nmap.File)
		if err != nil {
			log.Fatal(err)
		}
		nmapXML, err := nmap.Parse(bytes)
		if err != nil {
			log.Fatal(err)
		}
		for _, n := range nmapXML.Hosts {
			var h host
			h.Name = string(n.Hostnames[0].Name)
			for _, p := range n.Ports {
				h.Ports = append(h.Ports, int(p.PortId))
			}
			hosts = append(hosts, h)
		}
	} else {
		fmt.Println("Use hosts from config.json")
		hosts = config.Hosts
	}
	fmt.Print("Connect to mongodb...")
	collection, err := dbConnect(config.Db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Drop collection...")
	if err := dbDrop(collection); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	fmt.Print("Start scan...")
	for _, h := range hosts {
		for _, p := range h.Ports {
			go getHTTP(h.Name, p)
		}
	}
	fmt.Println("OK!")
	fmt.Printf("Active gorutines %v\n", runtime.NumGoroutine())
	time.Sleep(1 * time.Second)
	wg.Wait()
	fmt.Println("Complete scan")
	fmt.Print("Retrive data from database...")
	filter := bson.M{"status": bson.M{"$ne": ""}}
	var results documents
	if err := results.Read(collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Println("Print ALL documents")
	fmt.Println("Count: ", len(results))
	for _, r := range results {
		fmt.Println(r.Method, r.Scheme, r.Host, http.StatusText(r.Status), r.Header.Get("Content-Type"))
	}
	fmt.Println("Print ONE document")
	var result document
	if err := result.Read(collection, bson.M{"name": bson.M{"$eq": "getinside.cloud"}, "port": bson.M{"$lt": 1024}}); err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Host)
	fmt.Println(string(result.Body))

}

func (c *configuration) Load(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, c); err != nil {
		return err
	}
	return nil
}

func getHTTP(name string, port int) error {
	wg.Add(1)
	defer wg.Done()
	url := fmt.Sprintf("http://%s:%d", name, port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Get(url)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(r.Body)
	d := document{
		Name:   name,
		Port:   port,
		URL:    url,
		Method: r.Request.Method,
		Scheme: r.Request.URL.Scheme,
		Host:   r.Request.Host,
		Status: r.StatusCode,
		Header: r.Header,
		Body:   body,
	}
	if err := d.Write(collection); err != nil {
		return err
	}
	return nil
}

func (d *document) Write(c *mongo.Collection) error {
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
func (d *document) Read(c *mongo.Collection, f bson.M) error {
	if err := c.FindOne(context.Background(), f).Decode(&d); err != nil {
		return err
	}
	return nil
}

func (d *documents) Read(c *mongo.Collection, f bson.M) error {
	cursor, err := c.Find(context.TODO(), f)
	if err != nil {
		return err
	}
	for cursor.Next(context.TODO()) {
		var result document
		if err := cursor.Decode(&result); err != nil {
			return err
		}
		*d = append(*d, result)
	}
	return nil
}

func dbDelete(c *mongo.Collection, filter bson.M) error {
	if _, err := c.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}
	return nil
}

func dbDrop(c *mongo.Collection) error {
	if err := c.Drop(context.TODO()); err != nil {
		return err
	}
	return nil
}

func dbConnect(d database) (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, err
	}
	collection = client.Database(d.Db).Collection(d.Coll)
	return collection, nil
}
