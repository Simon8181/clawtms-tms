package services

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type WeatherService struct{}

func NewWeatherService() *WeatherService {
	return &WeatherService{}
}

type weatherResponse struct {
	CurrentCondition []struct {
		TempF       string `json:"temp_F"`
		WeatherDesc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	} `json:"current_condition"`
}

func (s *WeatherService) GetWeather(zipCode string) (string, error) {
	url := "http://wttr.in/" + zipCode + "?format=j1"

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result weatherResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.CurrentCondition) > 0 {
		cond := result.CurrentCondition[0].WeatherDesc[0].Value
		icon := getWeatherIcon(cond)
		return icon + " " + result.CurrentCondition[0].TempF + "F", nil
	}

	return "", nil
}

func getWeatherIcon(condition string) string {
	condition = strings.ToLower(condition)
	if strings.Contains(condition, "sun") || strings.Contains(condition, "clear") {
		return "☀️"
	} else if strings.Contains(condition, "cloud") {
		return "☁️"
	} else if strings.Contains(condition, "rain") {
		return "🌧️"
	} else if strings.Contains(condition, "storm") || strings.Contains(condition, "thunder") {
		return "⛈️"
	} else if strings.Contains(condition, "snow") {
		return "❄️"
	} else if strings.Contains(condition, "fog") || strings.Contains(condition, "mist") {
		return "🌫️"
	}
	return "🌤️"
}
