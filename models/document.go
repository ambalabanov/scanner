package models

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Document struct {
	ID                primitive.ObjectID `bson:"_id" json:"id"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
	Domain            string             `bson:"domain" json:"domain"`
	URL               string             `bson:"url" json:"url"`
	Method            string             `bson:"method" json:"method"`
	Scheme            string             `bson:"scheme" json:"scheme"`
	Host              string             `bson:"host" json:"host,omitempty"`
	Status            int                `bson:"status" json:"status"`
	Header            http.Header        `bson:"header" json:"header"`
	Links             []string           `bson:"links" json:"links,omitempty"`
	Title             string             `bson:"title" json:"title,omitempty"`
	Forms             []Form             `bson:"forms" json:"forms,omitempty"`
	Scripts           []string           `bson:"scripts" json:"scripts,omitempty"`
	Subdomaintakeover string             `bson:"subdomaintakeover" json:"subdomaintakeover,omitempty"`
	CNAME             string             `bson:"cname" json:"cname"`
}
type Form struct {
	CSRF   bool    `bson:"form_csrf" json:"form_csrf"`
	Method string  `bson:"form_method" json:"form_method,omitempty"`
	Action string  `bson:"form_action" json:"form_action,omitempty"`
	Input  []Input `bson:"form_input" json:"form_input,omitempty"`
}
type Input struct {
	Type  string `bson:"input_type" json:"input_type,omitempty"`
	Name  string `bson:"input_name" json:"input_name,omitempty"`
	Value string `bson:"input_value" json:"input_value,omitempty"`
}
type Documents []Document
