package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
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
	config     configuration
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

func init() {
	var err error
	fmt.Print("Load config.json...")
	config, err = loadJSON("config.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if config.Nmap.Use {
		fmt.Printf("Use hosts from %s\n", config.Nmap.File)
		nmapXML, err := loadXML(config.Nmap.File)
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
	collection, err = dbConnect(config.Db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Drop collection...")
	err = dbDrop(collection)
	if err != nil {
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
	result, err := dbFind(filter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Println("Print results...")
	fmt.Println("Count: ", len(result))
	for _, r := range result {
		fmt.Println(r.Host, r.Header.Get("Server"))
	}
	for _, r := range result {
		fmt.Println(r.Method, r.Scheme, r.Host, http.StatusText(r.Status), r.Header.Get("Content-Type"))
	}

	var r document
	r.Read(collection, bson.M{"name": bson.M{"$eq": "getinside.cloud"}, "port": bson.M{"$lt": 1024}})
	fmt.Println(r.Host)
	fmt.Println(string(r.Body))
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
	err = d.Write(collection)
	if err != nil {
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
	err := c.FindOne(context.Background(), f).Decode(&d)
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

func dbDrop(c *mongo.Collection) error {
	err := c.Drop(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func dbFind(filter bson.M) ([]*document, error) {
	cursor, err := collection.Find(context.TODO(), filter)
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

func dbConnect(d database) (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return nil, err
	}
	collection = client.Database(d.Db).Collection(d.Coll)
	return collection, nil
}
