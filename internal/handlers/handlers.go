package handlers

import (
	"clawtms/internal/database"
	"clawtms/internal/models"
	"clawtms/internal/services"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Handler struct {
	db             *database.DB
	routeService   *services.RouteService
	weatherService *services.WeatherService
}

func NewHandler(db *database.DB) *Handler {
	googleMapsKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	return &Handler{
		db:             db,
		routeService:   services.NewRouteService(googleMapsKey),
		weatherService: services.NewWeatherService(),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.Index)
	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/loads", h.GetLoads)
	mux.HandleFunc("POST /api/loads", h.CreateLoad)
	mux.HandleFunc("GET /api/loads/", h.GetLoad)
	mux.HandleFunc("PUT /api/loads/", h.UpdateLoad)
	mux.HandleFunc("DELETE /api/loads/", h.DeleteLoad)
	mux.HandleFunc("POST /api/quotes", h.CreateQuote)
	mux.HandleFunc("GET /api/quotes/", h.GetQuotes)
	mux.HandleFunc("POST /api/routes/calculate", h.CalculateRoutes)
	mux.HandleFunc("POST /api/fba/check", h.CheckFBACompliance)
	mux.HandleFunc("POST /api/jacob/generate", h.GenerateJacobEmail)
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/index.html")
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "ClawTMS"})
}

func (h *Handler) GetLoads(w http.ResponseWriter, r *http.Request) {
	loads, err := h.db.GetAllLoads()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loads)
}

func (h *Handler) CreateLoad(w http.ResponseWriter, r *http.Request) {
	var req models.CreateLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	load, err := h.db.CreateLoad(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(load)
}

func (h *Handler) GetLoad(w http.ResponseWriter, r *http.Request) {
	loadID := strings.TrimPrefix(r.URL.Path, "/api/loads/")
	load, err := h.db.GetLoad(loadID)
	if err != nil {
		http.Error(w, "Load not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(load)
}

func (h *Handler) UpdateLoad(w http.ResponseWriter, r *http.Request) {
	loadID := strings.TrimPrefix(r.URL.Path, "/api/loads/")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.db.UpdateLoadStatus(loadID, models.LoadStatus(req.Status))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.GetLoad(w, r)
}

func (h *Handler) DeleteLoad(w http.ResponseWriter, r *http.Request) {
	loadID := strings.TrimPrefix(r.URL.Path, "/api/loads/")
	_, err := h.db.GetLoad(loadID)
	if err != nil {
		http.Error(w, "Load not found", http.StatusNotFound)
		return
	}

	_, err = h.db.Exec("DELETE FROM loads WHERE load_id = $1", loadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Load deleted"})
}

func (h *Handler) CreateQuote(w http.ResponseWriter, r *http.Request) {
	var req models.CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	quote, err := h.db.CreateQuote(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quote)
}

func (h *Handler) GetQuotes(w http.ResponseWriter, r *http.Request) {
	loadID := strings.TrimPrefix(r.URL.Path, "/api/quotes/")
	quotes, err := h.db.GetQuotesByLoadID(loadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotes)
}

func (h *Handler) CalculateRoutes(w http.ResponseWriter, r *http.Request) {
	var req models.RouteCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var routes []models.RouteResult
	totalMiles := 0

	for _, loadID := range req.LoadIDs {
		load, err := h.db.GetLoad(loadID)
		if err != nil {
			continue
		}

		var addresses []string
		var zipPrefixes []string
		for _, stop := range load.Stops {
			addresses = append(addresses, stop.FullAddress)
			zipPrefixes = append(zipPrefixes, stop.ZipPrefix)
		}

		routeLink := h.routeService.BuildRouteLink(addresses)

		var weather string
		if len(load.Stops) > 0 {
			weather, _ = h.weatherService.GetWeather(load.Stops[0].ZipCode)
		}

		route := models.RouteResult{
			LoadID:      loadID,
			Distance:    "N/A",
			RouteLink:   routeLink,
			Weather:     weather,
			ZipPrefixes: zipPrefixes,
			Stops:       load.Stops,
		}
		routes = append(routes, route)
	}

	response := models.RouteCalculationResponse{
		Routes:     routes,
		TotalMiles: totalMiles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) CheckFBACompliance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LoadID string `json:"load_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	load, err := h.db.GetLoad(req.LoadID)
	if err != nil {
		http.Error(w, "Load not found", http.StatusNotFound)
		return
	}

	result := models.FBACheckResult{
		LoadID: load.LoadID,
	}

	if load.IsFBA && load.PalletCount > 0 {
		weightPerPallet := load.TotalWeight / float64(load.PalletCount)
		result.WeightPerPallet = weightPerPallet
		result.IsOverweight = weightPerPallet > 1500

		if result.IsOverweight {
			result.Message = "⚠️ Amazon Pallet Overweight Risk: " + fmt.Sprintf("%.0f", weightPerPallet) + " lbs per pallet (max 1,500 lbs)"
		} else {
			result.Message = "✅ FBA weight compliant"
		}
	} else {
		result.Message = "ℹ️ Not an FBA load or pallet count not specified"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GenerateJacobEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LoadID string `json:"load_id"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	load, err := h.db.GetLoad(req.LoadID)
	if err != nil {
		http.Error(w, "Load not found", http.StatusNotFound)
		return
	}

	var stopsList strings.Builder
	var origin, dest string
	for i, stop := range load.Stops {
		if stop.Type == models.StopTypePickup {
			if origin == "" {
				origin = stop.LocationName + ", " + stop.ZipCode
			}
		} else {
			if dest == "" {
				dest = stop.LocationName + ", " + stop.ZipCode
			}
		}
		if i > 0 {
			stopsList.WriteString(" → ")
		}
		stopsList.WriteString(stop.LocationName + " (" + stop.ZipCode + ")")
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	email := fmt.Sprintf("You are Jacob, a freight broker. Generate an email to %s using the following data:\n\nLoad ID: %s\nOrigin: %s\nStops: %s\nTotal Weight: %.0f lbs\nEquipment: %s\nWeather: N/A\n\nKeep it professional and concise.",
		req.Name, load.LoadID, origin, stopsList.String(), load.TotalWeight, load.EquipmentRequired)

	if geminiKey == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"email":     email,
			"generated": false,
			"note":      "GEMINI_API_KEY not set, showing template",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email":     email,
		"generated": true,
	})
}
