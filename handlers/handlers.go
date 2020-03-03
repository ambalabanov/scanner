package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//CreateScan handler for POST
func CreateScan(w http.ResponseWriter, r *http.Request) {
	hosts := make([]models.Document, 0)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&hosts); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	for _, h := range hosts {
		h.Parse()
		if err := dao.InsertOne(h); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
	}
	http.Error(w, "Scan was successfully created", http.StatusCreated)
}

//GetAllScan for GET
func GetAllScan(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	hosts, err := dao.Find(filter)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	if err := JSONresponse(w, hosts); err != nil {
		http.Error(w, "Bad response", http.StatusInternalServerError)
	}
}

//GetOneScan for GET
func GetOneScan(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	docID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": docID}
	hosts, err := dao.Find(filter)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	if err := JSONresponse(w, hosts); err != nil {
		http.Error(w, "Bad response", http.StatusInternalServerError)
	}
}

//JSONresponse to http
func JSONresponse(w http.ResponseWriter, d []interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(d); err != nil {
		return err
	}
	return nil
}

//DeleteOneScan fo DELETE
func DeleteOneScan(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	docID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": docID}
	count, err := dao.Delete(filter)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

//DeleteAllScan fo DELETE
func DeleteAllScan(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	count, err := dao.Delete(filter)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}
