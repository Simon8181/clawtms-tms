package services

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type RouteService struct {
	googleMapsAPIKey string
}

func NewRouteService(apiKey string) *RouteService {
	return &RouteService{googleMapsAPIKey: apiKey}
}

type DistanceMatrixResponse struct {
	Rows []struct {
		Elements []struct {
			Distance struct {
				Text  string `json:"text"`
				Value int    `json:"value"`
			} `json:"distance"`
			Duration struct {
				Text  string `json:"text"`
				Value int    `json:"value"`
			} `json:"duration"`
			Status string `json:"status"`
		} `json:"elements"`
	} `json:"rows"`
}

func (s *RouteService) GetDistanceMatrix(origins, destinations string) (*DistanceMatrixResponse, error) {
	url := "https://maps.googleapis.com/maps/api/distancematrix/json?origins=" + origins + "&destinations=" + destinations + "&key=" + s.googleMapsAPIKey

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result DistanceMatrixResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *RouteService) BuildRouteLink(addresses []string) string {
	if len(addresses) < 2 {
		return ""
	}
	var encoded []string
	for _, addr := range addresses {
		encoded = append(encoded, strings.ReplaceAll(addr, " ", "+"))
	}
	link := "https://www.google.com/maps/dir/" + strings.Join(encoded, "/")
	return link
}

func (s *RouteService) CalculateTotalMiles(distanceValue int) int {
	return distanceValue / 1609
}
