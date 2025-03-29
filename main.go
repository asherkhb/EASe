package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/gorilla/mux"
)

type Config struct {
	Port int `env:"EASE_PORT" envDefault:"8080"`
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the home page!\n")
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the login page!\n")
}

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", home).Methods("GET")
	r.HandleFunc("/login", login).Methods("GET")

	fmt.Println("Listening on port:", cfg.Port)
	log.
		err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
