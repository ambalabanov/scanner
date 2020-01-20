package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ambalabanov/go-nmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	wg         sync.WaitGroup
	collection *mongo.Collection
	hosts      documents
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
	URI   string `json:"uri"`
	Db    string `json:"db"`
	Coll  string `json:"coll"`
	Empty bool   `json:"empty"`
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
	Links  []string    `bson:"links"`
}
type documents []document

func init() {
	fmt.Print("Load config.json...")
	var config configuration
	if err := config.Load("config.json"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Connect to mongodb...")
	var err error
	if collection, err = dbConnect(config.Db); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if config.Db.Empty {
		fmt.Print("Drop collection...")
		if err := dbDrop(collection); err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK!")
	}
	fmt.Print("Load hosts...")
	if err := hosts.Load(&config); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	fmt.Print("Scan...")
	hosts.Scan()
	fmt.Println("OK!")
	fmt.Print("Retrive data from database...")
	filter := bson.M{"status": bson.M{"$ne": ""}}
	var results documents
	if err := results.Read(collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Parse URL's...")
	results.Parse()
	fmt.Println("OK!")
	fmt.Println("Print ONE document")
	var result document
	filter = bson.M{"name": "scanme.nmap.org", "port": bson.M{"$eq": 80}, "body": bson.M{"$ne": nil}}
	if err := result.Read(collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Links)

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

func (d *documents) Load(config *configuration) error {
	for _, s := range []string{"http", "https"} {
		if config.Nmap.Use {
			bytes, err := ioutil.ReadFile(config.Nmap.File)
			if err != nil {
				return err
			}
			nmapXML, err := nmap.Parse(bytes)
			if err != nil {
				return err
			}
			for _, n := range nmapXML.Hosts {
				var doc document
				for _, p := range n.Ports {
					doc.Name = string(n.Hostnames[0].Name)
					doc.Port = int(p.PortId)
					doc.Scheme = s
					*d = append(*d, doc)
				}
			}
		} else {
			for _, n := range config.Hosts {
				var doc document
				for _, p := range n.Ports {
					doc.Name = n.Name
					doc.Port = p
					doc.Scheme = s
					*d = append(*d, doc)
				}
			}
		}
	}
	return nil
}

func (d document) Scan() error {
	defer wg.Done()
	url := fmt.Sprintf("%s://%s:%d", d.Scheme, d.Name, d.Port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Head(url)
	if err != nil {
		return err
	}
	d.URL = url
	d.Method = r.Request.Method
	d.Scheme = r.Request.URL.Scheme
	d.Host = r.Request.Host
	d.Status = r.StatusCode
	d.Header = r.Header
	if err := d.Write(collection); err != nil {
		return err
	}
	return nil
}

func (d documents) Scan() error {
	for _, doc := range d {
		wg.Add(1)
		go doc.Scan()
	}
	wg.Wait()
	return nil
}

func (d documents) Parse() error {
	for _, doc := range d {
		wg.Add(1)
		go doc.Parse()
	}
	wg.Wait()
	return nil
}

func (d document) Parse() error {
	defer wg.Done()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Get(d.URL)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	d.Body = body
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	defer r.Body.Close()
	links := parseLinks(r.Body)
	d.Links = links
	if err := d.Write(collection); err != nil {
		return err
	}
	return nil
}

func parseLinks(b io.Reader) []string {
	var links []string
	doc := html.NewTokenizer(b)
	for tokenType := doc.Next(); tokenType != html.ErrorToken; {
		token := doc.Token()
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.A {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
		tokenType = doc.Next()
		continue
	}
	return links
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

func (d *documents) Write(c *mongo.Collection) error {
	docs := *d
	for _, doc := range docs {
		err := doc.Write(c)
		if err != nil {
			return err
		}
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
	client, _ := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, err
	}
	return client.Database(d.Db).Collection(d.Coll), nil
}
