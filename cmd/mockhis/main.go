package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Fixture patients keyed by national_id or passport_id.
// Reserved ids:
//   - "500ERROR"     → HTTP 500
//   - "BADJSON"      → undecodable body
//   - "ODDGENDER"    → 200 with gender outside {M,F}
var fixtures = map[string]map[string]any{
	"1100700123456": {
		"first_name_th":  "สมชาย",
		"middle_name_th": nil,
		"last_name_th":   "ใจดี",
		"first_name_en":  "Somchai",
		"middle_name_en": nil,
		"last_name_en":   "Jaidee",
		"date_of_birth":  "1990-01-15",
		"patient_hn":     "HN-10001",
		"national_id":    "1100700123456",
		"passport_id":    nil,
		"phone_number":   "0812345678",
		"email":          "somchai@example.com",
		"gender":         "M",
	},
	"A1234567": {
		"first_name_th":  nil,
		"middle_name_th": nil,
		"last_name_th":   nil,
		"first_name_en":  "Alice",
		"middle_name_en": "Q",
		"last_name_en":   "Wong",
		"date_of_birth":  "1985-06-20",
		"patient_hn":     "HN-20002",
		"national_id":    nil,
		"passport_id":    "A1234567",
		"phone_number":   "+66812345679",
		"email":          "alice.wong@example.com",
		"gender":         "F",
	},
	"ODDGENDER": {
		"first_name_th":  "ทดสอบ",
		"middle_name_th": nil,
		"last_name_th":   "เพศ",
		"first_name_en":  "Test",
		"middle_name_en": nil,
		"last_name_en":   "Gender",
		"date_of_birth":  "2000-01-01",
		"patient_hn":     "HN-99999",
		"national_id":    "ODDGENDER",
		"passport_id":    nil,
		"phone_number":   nil,
		"email":          nil,
		"gender":         "X",
	},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/patient/search/", handleSearch)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("hospital-a mock listening on :%s", port)
	if operationError := srv.ListenAndServe(); operationError != nil && operationError != http.ErrServerClosed {
		log.Fatalf("server error: %v", operationError)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/patient/search/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	switch id {
	case "500ERROR":
		http.Error(w, "internal upstream error", http.StatusInternalServerError)
		return
	case "BADJSON":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not-json`))
		return
	}

	patient, ok := fixtures[id]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(patient)
}
