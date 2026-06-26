package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/theme.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../src/theme.css")
	})
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}