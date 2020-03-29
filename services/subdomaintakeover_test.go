package services

import "testing"

func TestSubCheck(t *testing.T) {
	type args struct {
		body []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{[]byte("")}, "Not vulnerable"},
		{"test", args{[]byte("testtesttest")}, "Not vulnerable"},
		{"Bitbucket", args{[]byte("Repository not found")}, "Possible vulnerable"},
		{"Github", args{[]byte("There isn't a Github Pages site here.")}, "Possible vulnerable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubCheck(tt.args.body); got != tt.want {
				t.Errorf("SubCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCNAME(t *testing.T) {
	type args struct {
		domain string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{""}, ""},
		{"test", args{"testtesttest"}, ""},
		{"Bitbucket", args{"testtesttest.bitbucket.org"}, "bitbucket.org."},
		{"Github", args{"testtesttest.github.com"}, "github.github.io."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCNAME(tt.args.domain); got != tt.want {
				t.Errorf("getCNAME() = %v, want %v", got, tt.want)
			}
		})
	}
}
