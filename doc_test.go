package otomatik

import (
	"fmt"
	"log"
	"net/http"
)

// This is the simplest way for HTTP servers to use this package.
// Call HTTPS() with your domain names and your handler (or nil
// for the http.DefaultMux), and otomatik will do the rest.
func ExampleHTTPS() {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Hello, HTTPS visitor!")
	})

	err := HTTPS([]string{"example.com", "www.example.com"}, nil)
	if err != nil {
		log.Fatal(err)
	}
}
