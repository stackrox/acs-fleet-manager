package shared

import (
	"encoding/json"
	"net/http"
)

// WriteJSONResponse writes a json HTTP response of the given HTTP status code and response payload
func WriteJSONResponse(w http.ResponseWriter, code int, payload interface{}) {
	WriteStreamJSONResponseWithContentType(w, code, payload, "application/json")
}

// WriteStreamJSONResponseWithContentType WriteJSONResponse writes a HTTP response of the given HTTP status code and response payload with a given content type
func WriteStreamJSONResponseWithContentType(w http.ResponseWriter, code int, payload interface{}, contentType string) {
	w.Header().Set("Content-Type", contentType)
	// By default, decide whether or not a cache is usable based on the matching of the JWT
	// For example, this will keep caches from being used in the same browser if two users were to log in back to back
	w.Header().Set("Vary", "Authorization")
	w.WriteHeader(code)

	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
		// In case we need to debug response encoding... use this instead:
		// response, _ := json.Marshal(payload)
		// fmt.Println(string(response))
		// _, _ = w.Write(response)
	}
}
