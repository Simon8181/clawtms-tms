package main

import (
	"clawtms/internal/database"
	"clawtms/internal/handlers"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := database.Connect()
	if err != nil {
		log.Printf("Warning: Could not connect to database: %v", err)
		log.Println("Running in demo mode without database")
		db = nil
	} else {
		if err := db.InitSchema(); err != nil {
			log.Printf("Warning: Could not initialize schema: %v", err)
		}
	}

	handler := handlers.NewHandler(db)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	log.Printf("Starting ClawTMS Server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
