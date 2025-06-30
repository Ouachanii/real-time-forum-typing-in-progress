package posts

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})

	if r.Method != http.MethodGet {
		response["error"] = "Invalid request method."
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.URL.Path != "/" {
		ErrorPage(w, 404, "Page Not Found")
		return
	}

	tmpl, err := template.ParseFiles("./frontend/html/index.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering posts", http.StatusInternalServerError)
		return
	}
}
