package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Forecast contém os dados de previsão do tempo
type Forecast struct {
	Temperature float64
	Condition   string  // Ex: "Clouds", "Rain", "Clear"
	Description string  // Ex: "partly cloudy" / "parcialmente nublado"
	WindSpeed   float64 // km/h
}

// Estruturas para decodificar a resposta de geocoding do Open-Meteo
type geocodingResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Country   string  `json:"country"`
	} `json:"results"`
}

// Estruturas para decodificar a resposta de weather do Open-Meteo
type weatherResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		WindSpeed   float64 `json:"windspeed"`
		WeatherCode int     `json:"weathercode"`
	} `json:"current_weather"`
}

type Client struct {
	httpClient *http.Client
	latitude   float64
	longitude  float64
	resolved   bool
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetCurrentWeather(city string) (*Forecast, error) {
	// Resolve city to coordinates on first call
	if !c.resolved {
		if err := c.resolveCity(city); err != nil {
			return nil, fmt.Errorf("failed to geocode city %q: %w", city, err)
		}
		c.resolved = true
	}

	// Fetch current weather from Open-Meteo
	weatherURL := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true",
		c.latitude, c.longitude,
	)

	resp, err := c.httpClient.Get(weatherURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status: %s", resp.Status)
	}

	var data weatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode weather response: %w", err)
	}

	condition, description := weatherCodeToCondition(data.CurrentWeather.WeatherCode)

	return &Forecast{
		Temperature: data.CurrentWeather.Temperature,
		Condition:   condition,
		Description: description,
		WindSpeed:   data.CurrentWeather.WindSpeed,
	}, nil
}

func (c *Client) resolveCity(city string) error {
	// Strip country code if present (e.g. "Sao Paulo,BR" -> "Sao Paulo")
	cleanCity := city
	if idx := len(city) - 1; idx > 0 {
		for i := len(city) - 1; i >= 0; i-- {
			if city[i] == ',' {
				cleanCity = city[:i]
				break
			}
		}
	}

	geocodeURL := fmt.Sprintf(
		"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
		url.QueryEscape(cleanCity),
	)

	resp, err := c.httpClient.Get(geocodeURL)
	if err != nil {
		return fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("geocoding API returned status: %s", resp.Status)
	}

	var data geocodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	if len(data.Results) == 0 {
		return fmt.Errorf("city %q not found", city)
	}

	c.latitude = data.Results[0].Latitude
	c.longitude = data.Results[0].Longitude
	return nil
}

// weatherCodeToCondition maps WMO weather codes to condition strings
// Uses system LANG to translate descriptions.
func weatherCodeToCondition(code int) (condition, description string) {
	lang := detectLanguage()
	switch {
	case code == 0:
		return "Clear", translate(lang, "clear sky")
	case code == 1:
		return "Clear", translate(lang, "mainly clear")
	case code == 2:
		return "Clouds", translate(lang, "partly cloudy")
	case code == 3:
		return "Clouds", translate(lang, "overcast")
	case code >= 45 && code <= 48:
		return "Fog", translate(lang, "fog")
	case code >= 51 && code <= 55:
		return "Drizzle", translate(lang, "drizzle")
	case code >= 56 && code <= 57:
		return "Drizzle", translate(lang, "freezing drizzle")
	case code >= 61 && code <= 65:
		return "Rain", translate(lang, "rain")
	case code >= 66 && code <= 67:
		return "Rain", translate(lang, "freezing rain")
	case code >= 71 && code <= 77:
		return "Snow", translate(lang, "snow")
	case code >= 80 && code <= 82:
		return "Rain", translate(lang, "rain showers")
	case code >= 85 && code <= 86:
		return "Snow", translate(lang, "snow showers")
	case code == 95:
		return "Thunderstorm", translate(lang, "thunderstorm")
	case code >= 96 && code <= 99:
		return "Thunderstorm", translate(lang, "thunderstorm with hail")
	default:
		return "Unknown", translate(lang, "unknown")
	}
}

// detectLanguage returns the 2-letter language code from system locale.
func detectLanguage() string {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LC_ALL")
	}
	if lang == "" {
		lang = os.Getenv("LANGUAGE")
	}
	if lang == "" {
		return "en"
	}
	// LANG format: "pt_BR.UTF-8" -> "pt"
	lang = strings.ToLower(lang)
	if idx := strings.IndexAny(lang, "_."); idx > 0 {
		lang = lang[:idx]
	}
	return lang
}

// translate returns the localized version of a weather description.
func translate(lang, english string) string {
	if lang == "en" {
		return english
	}
	if translations, ok := weatherTranslations[lang]; ok {
		if t, ok := translations[english]; ok {
			return t
		}
	}
	return english
}

var weatherTranslations = map[string]map[string]string{
	"pt": {
		"clear sky":              "céu limpo",
		"mainly clear":           "predominantemente limpo",
		"partly cloudy":          "parcialmente nublado",
		"overcast":               "nublado",
		"fog":                    "neblina",
		"drizzle":                "garoa",
		"freezing drizzle":       "garoa congelante",
		"rain":                   "chuva",
		"freezing rain":          "chuva congelante",
		"snow":                   "neve",
		"rain showers":           "pancadas de chuva",
		"snow showers":           "pancadas de neve",
		"thunderstorm":           "tempestade",
		"thunderstorm with hail": "tempestade com granizo",
		"unknown":                "desconhecido",
	},
	"es": {
		"clear sky":              "cielo despejado",
		"mainly clear":           "mayormente despejado",
		"partly cloudy":          "parcialmente nublado",
		"overcast":               "nublado",
		"fog":                    "niebla",
		"drizzle":                "llovizna",
		"freezing drizzle":       "llovizna helada",
		"rain":                   "lluvia",
		"freezing rain":          "lluvia helada",
		"snow":                   "nieve",
		"rain showers":           "chubascos",
		"snow showers":           "chubascos de nieve",
		"thunderstorm":           "tormenta",
		"thunderstorm with hail": "tormenta con granizo",
		"unknown":                "desconocido",
	},
}
