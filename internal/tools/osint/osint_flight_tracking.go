package osint

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// FlightData represents tracked flight information.
type FlightData struct {
	FlightNumber string  `json:"flight_number"`
	Airline      string  `json:"airline"`
	Origin       string  `json:"origin"`
	Destination  string  `json:"destination"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Altitude     float64 `json:"altitude"` // feet
	Speed        float64 `json:"speed"`    // knots
	Status       string  `json:"status"`
	IsSynthetic  bool    `json:"is_synthetic"`
	GeneratedAt  string  `json:"generated_at"`
}

type FlightTrackingTool struct {
	broker *Shadowbroker
}

func NewFlightTrackingTool(broker *Shadowbroker) *FlightTrackingTool {
	return &FlightTrackingTool{broker: broker}
}

// Track returns tracking data for a flight identified by flight number.
func (t *FlightTrackingTool) Track(ctx context.Context, flightNumber string) (map[string]interface{}, error) {
	if flightNumber == "" {
		return nil, fmt.Errorf("flight_number is required")
	}
	return generateFlightData(flightNumber), nil
}

// Execute implements the JSON→JSON tool interface.
func (t *FlightTrackingTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		FlightNumber string `json:"flight_number"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.FlightNumber == "" {
		return "", fmt.Errorf("flight_number is required")
	}
	result, err := t.Track(ctx, args.FlightNumber)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

func (t *FlightTrackingTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_flight_tracking",
		Name:         "osint_flight_tracking",
		Description:  "Flight tracking via Shadowbroker (beta) | is_synthetic=true | privacy-preserving",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

var airlines = []string{
	"Delta Air Lines", "United Airlines", "American Airlines",
	"Lufthansa", "Emirates", "British Airways", "Air France", "Qatar Airways",
}

var airports = []string{
	"JFK", "LAX", "LHR", "CDG", "DXB", "HND", "FRA", "SIN", "AMS", "IST",
}

var flightStatuses = []string{"scheduled", "en_route", "landed", "delayed", "cancelled"}

func generateFlightData(flightNumber string) map[string]interface{} {
	seed := int64(hashString(flightNumber))
	rng := rand.New(rand.NewSource(seed))

	origin := airports[rng.Intn(len(airports))]
	dest := airports[(int(hashString(flightNumber))+1)%len(airports)]
	for dest == origin {
		dest = airports[rng.Intn(len(airports))]
	}

	lat := (rng.Float64() * 180) - 90
	lon := (rng.Float64() * 360) - 180
	altitude := 25000 + rng.Float64()*15000
	speed := 350 + rng.Float64()*200
	status := flightStatuses[rng.Intn(len(flightStatuses))]

	return map[string]interface{}{
		"flight_number": flightNumber,
		"airline":       airlines[int(hashString(flightNumber))%len(airlines)],
		"origin":        origin,
		"destination":   dest,
		"latitude":      roundFloat(lat, 4),
		"longitude":     roundFloat(lon, 4),
		"altitude":      roundFloat(altitude, 0),
		"speed":         roundFloat(speed, 1),
		"status":        status,
		"is_synthetic":  true,
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
	}
}
