package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the home page!\n")
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the login page!\n")
}

func main() {
	port := 8080

	r := mux.NewRouter()

	r.HandleFunc("/", home).Methods("GET")
	r.HandleFunc("/login", login).Methods("GET")

	fmt.Println("Listening on port:", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), r)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
