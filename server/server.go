package main

import (
	"fmt"
	"log"
	"net/http"
)

func trackHandler(w http.ResponseWriter, r *http.Request) {
	step := r.URL.Query().Get("step")
	log.Printf("Шаг: %s\n", step)
	fmt.Fprintf(w, "Шаг принят: %s\n", step)
}

func main() {
	http.HandleFunc("/track", trackHandler)
	log.Println("Сервер трекинга запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
