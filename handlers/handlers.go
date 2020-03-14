package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"github.com/ambalabanov/scanner/services"
	"github.com/gorilla/mux"
)

//CreateScan handler for POST
func CreateScan(w http.ResponseWriter, r *http.Request) {
	var hosts models.Documents
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&hosts); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	go services.Parse(hosts)
	http.Error(w, "Scan was successfully created", http.StatusCreated)
}

//GetAllScan for GET
func GetAllScan(w http.ResponseWriter, r *http.Request) {
	hosts, err := dao.FindAll()
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
	hosts, err := dao.FindOne(params["id"])
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
		return
	}
}

//JSONresponse to http
func JSONresponse(w http.ResponseWriter, d []models.Document) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(d)
	return err
}

//DeleteOneScan for DELETE
func DeleteOneScan(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	count, err := dao.DeleteOne(params["id"])
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

//DeleteAllScan for DELETE
func DeleteAllScan(w http.ResponseWriter, r *http.Request) {
	count, err := dao.DeleteAll()
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}
