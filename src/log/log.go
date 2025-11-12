package log

import (
	"fmt"
	"net/http"
)

// * здесь мог быть мой логгер, но chi предоставляет свой:)
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("request - ", r.Body, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
