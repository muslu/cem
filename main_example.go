package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
	Data    string `json:"data"`
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Only GET method is allowed",
		})
		return
	}

	// JSON cevabı hazırla
	response := Response{
		Message: "Başarılı",
		Status:  200,
		Data:    "Go HTTP Server çalışıyor!",
	}

	// Header'ı JSON olarak ayarla
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// JSON encode et ve cevap gönder
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Handler'ı kayıt et
	http.HandleFunc("/api/test", handleRequest)

	// Sunucuyu başlat
	address := ":8080"
	fmt.Printf("Server http://localhost%s adresinde çalışıyor\n", address)
	fmt.Println("Test etmek için: curl http://localhost:8080/api/test")

	log.Fatal(http.ListenAndServe(address, nil))
}
