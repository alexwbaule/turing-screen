package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Forecast contém apenas os dados que nos interessam
type Forecast struct {
	Temperature float64
	Condition   string // Ex: "Clouds", "Rain", "Clear"
	Description string // Ex: "scattered clouds"
}

// Estruturas para decodificar a resposta JSON da API
type apiResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
}

type Client struct {
	httpClient *http.Client
	apiKey     string
}

func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiKey:     apiKey,
	}
}

func (c *Client) GetCurrentWeather(city string) (*Forecast, error) {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", city, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned non-200 status: %s", resp.Status)
	}

	var data apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode weather response: %w", err)
	}

	if len(data.Weather) == 0 {
		return nil, fmt.Errorf("no weather data in API response")
	}

	forecast := &Forecast{
		Temperature: data.Main.Temp,
		Condition:   data.Weather[0].Main,
		Description: data.Weather[0].Description,
	}

	return forecast, nil
}
