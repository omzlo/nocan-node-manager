package nocan

import (
    "encoding/json"
    "net/http"
)

func RenderJSON(w http.ResponseWriter, v interface{}) bool {
    js, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return false
    }
    if _, err := w.Write(js); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return false
    }
    return true
}

