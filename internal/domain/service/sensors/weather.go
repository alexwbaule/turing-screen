package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/resource/weather"
)

type WeatherSensor struct {
	log           *logger.Logger
	jobs          chan<- command.Command
	builder       *renderer.Builder
	p             *command.UpdatePayload
	weatherClient *weather.Client
	city          string
}

// Construtor para o modo estático
func NewWeatherSensor(l *logger.Logger, j chan<- command.Command, b *renderer.Builder, p *command.UpdatePayload, client *weather.Client, city string) *WeatherSensor {
	return &WeatherSensor{
		log:           l.With("runner", "weather_sensor"),
		jobs:          j,
		builder:       b,
		p:             p,
		weatherClient: client,
		city:          city,
	}
}

func (g *WeatherSensor) Run(ctx context.Context, e *theme.Weather, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := g.getWeather(ctx, e); err != nil {
		g.log.Errorf("failed to get initial weather: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping WeatherSensor")
			return ctx.Err()
		case <-ticker.C:
			if err := g.getWeather(ctx, e); err != nil {
				g.log.Errorf("failed to get weather update: %v", err)
			}
		}
	}
}

func (g *WeatherSensor) getWeather(ctx context.Context, e *theme.Weather) error {
	var payloads []*command.UpdatePayload

	forecast, err := g.weatherClient.GetCurrentWeather(g.city)
	if err != nil {
		return err
	}

	g.log.Infof("weather: %.1f°C, %s (%s)", forecast.Temperature, forecast.Condition, forecast.Description)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Exibe a temperatura (se configurado no tema)
	if e.Temperature != nil {
		if e.Temperature.Text != nil && e.Temperature.Text.Show {
			img, x, y := BuildText(g.builder, forecast.Temperature, "%.0f", "°C", e.Temperature.Text, SizeTemp)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}

	// Exibe a condição do tempo (se configurado no tema)
	if e.Condition != nil && e.Condition.Show {
		img, x, y := BuildText(g.builder, forecast.Description, "", "", e.Condition, SizeDefault)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping weather")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}

	return nil
}
