package ezrpc

import (
	"encoding/json"
	"log"
	"net/http"
)

// ServerError は、サーバ側でエラーが発生したことを表す。
type ServerError struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
}

func (e *ServerError) write(w http.ResponseWriter, r *http.Request) {
	if e.StatusCode == 0 {
		e.StatusCode = 500
	}

	w.WriteHeader(e.StatusCode)
	if e.Message == "" {
		e.Message = http.StatusText(e.StatusCode)
	}

	b, _ := json.Marshal(e)
	w.Write(b)
	log.Println(string(b))
}

func (e *ServerError) Error() string {
	return e.Message
}
