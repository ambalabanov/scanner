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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type configuration struct {
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
	Db   database `json:"database"`
	Nmap struct {
		Use  bool   `json:"use"`
		File string `json:"file"`
	} `json:"nmap"`
}
type host struct {
	Name  string `json:"name"`
	Ports []int  `json:"ports"`
}
type database struct {
	Use        bool   `json:"use"`
	URI        string `json:"uri"`
	Db         string `json:"db"`
	Coll       string `json:"coll"`
	Empty      bool   `json:"empty"`
	collection *mongo.Collection
}
type document struct {
	ID     primitive.ObjectID `bson:"_id"    json:"id"`
	Name   string             `bson:"name"   json:"name"`
	Port   int                `bson:"port"   json:"port"`
	URL    string             `bson:"url"    json:"url"`
	Method string             `bson:"method" json:"method"`
	Scheme string             `bson:"scheme" json:"scheme"`
	Host   string             `bson:"host"   json:"host"`
	Status int                `bson:"status" json:"status"`
	Header http.Header        `bson:"header" json:"-"`
	Body   []byte             `bson:"body"   json:"-"`
	Links  []string           `bson:"links"  json:"links"`
	Title  string             `bson:"title"  json:"title"`
}
type documents []document

func main() {
	log.Println("Load config")
	var config configuration
	if err := config.Load("config.json"); err != nil {
		log.Fatal(err)
	}
	log.Println("Connect to mongodb")
	db := config.Db
	if err := db.connect(); err != nil {
		log.Fatal(err)
	}
	var hosts documents
	log.Printf("Server starting on port %v...\n", config.Server.Port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var h []host
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&h); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		log.Println("Load hosts")
		hosts.Load(h)
		log.Println("Scan hosts")
		hosts.Scan()
		log.Println("Parse body")
		hosts.Parse()
		log.Println("Drop collection")
		if err := db.drop(); err != nil {
			log.Fatal(err)
		}
		log.Println("Write to database")
		if err := hosts.Write(db.collection); err != nil {
			log.Fatal(err)
		}
		log.Println("Read from database")
		hosts = documents{}
		filter := bson.M{}
		if err := hosts.Read(db.collection, filter); err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(&hosts); err != nil {
			log.Fatal(err)
		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", config.Server.Port), nil))

}

func (d *documents) Load(h []host) {
	for _, s := range []string{"http", "https"} {
		for _, n := range h {
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

func (d *documents) LoadNMAP(filename string) error {
	for _, s := range []string{"http", "https"} {
		bytes, err := ioutil.ReadFile(filename)
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
	}
	return nil
}

func (d document) Scan(res chan document, wg *sync.WaitGroup) error {
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
	res <- d
	return nil
}

func (d *documents) Scan() error {
	var wg sync.WaitGroup
	var dd documents
	res := make(chan document, len(*d))
	for _, doc := range *d {
		wg.Add(1)
		go doc.Scan(res, &wg)
	}
	wg.Wait()
	for i, l := 0, len(res); i < l; i++ {
		dd = append(dd, <-res)
	}
	*d = dd
	return nil
}

func (d *documents) Parse() error {
	var wg sync.WaitGroup
	var dd documents
	res := make(chan document, len(*d))
	for _, doc := range *d {
		wg.Add(1)
		go doc.Parse(res, &wg)
	}
	wg.Wait()
	for i, l := 0, len(res); i < l; i++ {
		dd = append(dd, <-res)
	}
	*d = dd
	return nil
}

func (d document) Parse(res chan document, wg *sync.WaitGroup) error {
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
	d.parseLinks(ioutil.NopCloser(bytes.NewBuffer(body)))
	d.parseTitle(ioutil.NopCloser(bytes.NewBuffer(body)))
	d.Method = r.Request.Method
	res <- d
	return nil
}

func (d *document) parseLinks(b io.Reader) {
	var links []string
	tokenizer := html.NewTokenizer(b)
	for tokenType := tokenizer.Next(); tokenType != html.ErrorToken; {
		token := tokenizer.Token()
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.A {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
		tokenType = tokenizer.Next()
	}
	d.Links = links
}

func (d *document) parseTitle(b io.Reader) {
	tokenizer := html.NewTokenizer(b)
	for tokenType := tokenizer.Next(); tokenType != html.ErrorToken; {
		token := tokenizer.Token()
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.Title {
				tokenType = tokenizer.Next()
				if tokenType == html.TextToken {
					d.Title = tokenizer.Token().Data
					break
				}
			}
		}
		tokenType = tokenizer.Next()
	}
}

func (d *document) Write(c *mongo.Collection) error {
	d.ID = primitive.NewObjectID()
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

func (d *database) delete(filter bson.M) error {
	if _, err := d.collection.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}
	return nil
}

func (d *database) drop() error {
	if err := d.collection.Drop(context.TODO()); err != nil {
		return err
	}
	return nil
}

func (d *database) connect() error {
	client, _ := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return err
	}
	d.collection = client.Database(d.Db).Collection(d.Coll)
	return nil
}
