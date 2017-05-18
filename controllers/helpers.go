package controllers

import (
	"net/http"
)

/** ACCEPT **/

func AcceptJSON(r *http.Request) bool {
	return r.Header.Get("Accept") == "application/json"
}

func AcceptHTML(r *http.Request) bool {
	return r.Header.Get("Accept") == "text/html"
}
