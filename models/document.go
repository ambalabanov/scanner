package models

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Document struct {
	ID        primitive.ObjectID `bson:"_id"        json:"id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	URL       string             `bson:"url"        json:"url"`
	Method    string             `bson:"method"     json:"method"`
	Scheme    string             `bson:"scheme"     json:"scheme"`
	Host      string             `bson:"host"       json:"host"`
	Status    int                `bson:"status"     json:"status"`
	Header    http.Header        `bson:"header"     json:"header"`
	Links     []string           `bson:"links"      json:"links"`
	Title     string             `bson:"title"      json:"title"`
	Forms     []Form             `bson:"forms"      json:"forms"`
	Scripts   []string           `bson:"scripts"      json:"scripts"`
}

type Form struct {
	CSRF   bool     `bson:"form_csrf" json:"form_csrf"`
	Method string   `bson:"form_method" json:"form_method"`
	Action string   `bson:"form_action" json:"form_action"`
	Input  []string `bson:"form_input" json:"form_input"`
}

type Documents []Document
