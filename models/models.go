package models

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type document struct {
	ID        primitive.ObjectID `bson:"_id"        json:"id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	URL       string             `bson:"url"        json:"url"`
	Method    string             `bson:"method"     json:"method"`
	Scheme    string             `bson:"scheme"     json:"scheme"`
	Host      string             `bson:"host"       json:"host"`
	Status    int                `bson:"status"     json:"status"`
	Header    http.Header        `bson:"header"     json:"header"`
	Body      []byte             `bson:"body"       json:"-"`
	Links     []string           `bson:"links"      json:"links"`
	Title     string             `bson:"title"      json:"title"`
}
