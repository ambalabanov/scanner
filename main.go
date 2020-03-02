package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type configuration struct {
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
	Database struct {
		URI   string `json:"uri"`
		Db    string `json:"db"`
		Coll  string `json:"coll"`
		Empty bool   `json:"empty"`
	} `json:"database"`
}
type hosts struct {
	Urls []string `json:"urls"`
}

type documents []models.Document

var configPath = flag.String("c", "config.json", "Path to config.json")
var config configuration
var collection *mongo.Collection

func init() {
	flag.Parse()
	log.Printf("Load config: %v", *configPath)
	if err := config.load(configPath); err != nil {
		log.Fatal(err)
	}
	log.Println("Connect to mongodb")
	var err error
	collection, err = dao.Connect(config.Database.URI, config.Database.Db, config.Database.Coll)
	if err != nil {
		log.Fatal(err)
	}
	if config.Database.Empty {
		log.Println("Drop collection")
		if err := dao.Drop(collection); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/api/scan", getAllScan).Methods("GET")
	router.HandleFunc("/api/scan/{id:[0-9a-fA-F]+}", getOneScan).Methods("GET")
	router.HandleFunc("/api/scan", deleteAllScan).Methods("DELETE")
	router.HandleFunc("/api/scan/{id:[0-9a-fA-F]+}", deleteOneScan).Methods("DELETE")
	router.HandleFunc("/api/scan", createScan).Methods("POST")
	log.Printf("Server starting on port %v...\n", config.Server.Port)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("localhost:%v", config.Server.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func (d *documents) load(h hosts) {
	for _, u := range h.Urls {
		var doc models.Document
		doc.URL = u
		doc.ID = primitive.NewObjectID()
		doc.CreatedAt = time.Now()
		*d = append(*d, doc)
	}
}

func getAllScan(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	hosts := documents{}
	log.Println("Read from database")
	if err := hosts.read(collection, filter); err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	if err := hosts.response(w); err != nil {
		http.Error(w, "Bad response", http.StatusInternalServerError)
	}
}

func getOneScan(w http.ResponseWriter, r *http.Request) {
	hosts := documents{}
	params := mux.Vars(r)
	id := params["id"]
	docID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": docID}
	log.Println("Read from database")
	if err := hosts.read(collection, filter); err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	if err := hosts.response(w); err != nil {
		http.Error(w, "Bad response", http.StatusInternalServerError)
	}
}

func deleteOneScan(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	docID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": docID}
	log.Println("Delete from database")
	count, err := dao.Delete(collection, filter)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

func deleteAllScan(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	log.Println("Delete from database")
	count, err := dao.Delete(collection, filter)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

func createScan(w http.ResponseWriter, r *http.Request) {
	var h hosts
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&h); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	log.Println("Load hosts")
	hosts := documents{}
	hosts.load(h)
	log.Println("Parse body")
	hosts.parse()
	log.Println("Write to database")
	if err := hosts.write(collection); err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	http.Error(w, "Scan was successfully created", http.StatusCreated)
}

func (d *documents) response(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(d); err != nil {
		return err
	}
	return nil
}

func (c *configuration) load(filename *string) error {
	bytes, err := ioutil.ReadFile(*filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, c); err != nil {
		return err
	}
	return nil
}

func (d *documents) parse() error {
	var wg sync.WaitGroup
	var dd documents
	res := make(chan models.Document, len(*d))
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

func (d *documents) write(c *mongo.Collection) error {
	docs := *d
	for _, doc := range docs {
		err := dao.InsertOne(c, doc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *documents) read(c *mongo.Collection, f bson.M) error {
	cursor, err := c.Find(context.TODO(), f)
	if err != nil {
		return err
	}
	for cursor.Next(context.TODO()) {
		var result models.Document
		if err := cursor.Decode(&result); err != nil {
			return err
		}
		*d = append(*d, result)
	}
	return nil
}
