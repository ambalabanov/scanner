package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"github.com/ambalabanov/scanner/services"
	"github.com/gorilla/mux"
)

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func CreateScan(w http.ResponseWriter, r *http.Request) {
	hosts := services.LoadD(r.Body)
	go services.ParseH(hosts)
	http.Error(w, "Scan was successfully created", http.StatusCreated)
}

func CreateParse(w http.ResponseWriter, r *http.Request) {
	var hosts models.Documents
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hosts)
	if err != nil {
		handleError(w, err)
	}
	go services.ParseH(hosts)
	http.Error(w, "Parse was successfully created", http.StatusCreated)
}

func GetAllParse(w http.ResponseWriter, _ *http.Request) {
	hosts, err := dao.FindAll()
	if err != nil {
		handleError(w, err)
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	err = JSONResponse(w, hosts)
	if err != nil {
		handleError(w, err)
	}
}

func GetOneParse(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	hosts, err := dao.FindOne(params["id"])
	if err != nil {
		handleError(w, err)
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	err = JSONResponse(w, hosts)
	if err != nil {
		handleError(w, err)
	}
}

func JSONResponse(w http.ResponseWriter, d []models.Document) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(d)
	return err
}

func DeleteOneParse(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	count, err := dao.DeleteOne(params["id"])
	if err != nil {
		handleError(w, err)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

func DeleteAllParse(w http.ResponseWriter, _ *http.Request) {
	count, err := dao.DeleteAll()
	if err != nil {
		handleError(w, err)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}
