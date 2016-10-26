package nocan

import (
	"encoding/json"
	"net/http"
	"pannetrat.com/nocan/clog"
)

func RenderJSON(w http.ResponseWriter, v interface{}) bool {
	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(js); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func LogHttpError(w http.ResponseWriter, err string, code int) {
	http.Error(w, err, code)
	if code < 500 {
		clog.Warning("Returned HTTP status %d: %s", code, err)
	} else {
		clog.Error("Returned HTTP status %d: %s", code, err)
	}
}
