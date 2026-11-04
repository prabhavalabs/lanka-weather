package weather

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

// Value is a number in a response: rounded to a tenth, null when missing.
type Value float64

// MarshalJSON writes the value to one decimal, or null for NaN.
func (value Value) MarshalJSON() ([]byte, error) {
	number := float64(value)
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return []byte("null"), nil
	}
	return strconv.AppendFloat(nil, math.Round(number*10)/10, 'f', -1, 64), nil
}

// Values is a list of Value with a compact encoding.
type Values []Value

// MarshalJSON writes the list without spaces.
func (values Values) MarshalJSON() ([]byte, error) {
	out := make([]byte, 0, 6*len(values)+2)
	out = append(out, '[')
	for index, value := range values {
		if index > 0 {
			out = append(out, ',')
		}
		encoded, _ := value.MarshalJSON()
		out = append(out, encoded...)
	}
	return append(out, ']'), nil
}

// variable maps a response field to a stored variable.
type variable struct {
	// Name is the response field.
	Name string
	// Current and Hourly are the stored variable names; Hourly is used for
	// a current value only when Current is empty.
	Current, Hourly string
	// Categorical values take the nearest node rather than interpolating.
	Categorical bool
	Unit        string
}

// currentVariables are the values of a current-conditions grid or point, in
// response order. UV index and precipitation probability are not current
// observations at all; they come from the forecast's hour nearest the
// current values' time.
var currentVariables = []variable{
	{Name: "temperatureC", Current: "temperature_2m", Hourly: "temperature_2m", Unit: "°C"},
	{Name: "apparentTemperatureC", Current: "apparent_temperature", Hourly: "apparent_temperature", Unit: "°C"},
	{Name: "relativeHumidityPct", Current: "relative_humidity_2m", Hourly: "relative_humidity_2m", Unit: "%"},
	{Name: "dewPointC", Current: openmeteo.VariableDewPoint, Hourly: openmeteo.VariableDewPoint, Unit: "°C"},
	{Name: "precipitationMm", Current: "precipitation", Hourly: "precipitation", Unit: "mm"},
	{Name: "precipitationProbabilityPct", Hourly: "precipitation_probability", Unit: "%"},
	{Name: "cloudCoverPct", Current: "cloud_cover", Hourly: "cloud_cover", Unit: "%"},
	{Name: "pressureHpa", Current: "pressure_msl", Hourly: "pressure_msl", Unit: "hPa"},
	{Name: "windSpeedKmh", Current: "wind_speed_10m", Hourly: "wind_speed_10m", Unit: "km/h"},
	{Name: "windDirectionDeg", Current: "wind_direction_10m", Hourly: "wind_direction_10m", Unit: "°"},
	{Name: "windGustsKmh", Current: "wind_gusts_10m", Hourly: "wind_gusts_10m", Unit: "km/h"},
	{Name: "weatherCode", Current: "weather_code", Hourly: "weather_code", Categorical: true, Unit: "wmo"},
	{Name: "uvIndex", Hourly: "uv_index", Unit: ""},
	{Name: "isDay", Current: openmeteo.VariableIsDay, Hourly: openmeteo.VariableIsDay, Categorical: true, Unit: "0/1"},
}

// Units lists the unit of every variable in responses.
func Units() map[string]string {
	units := map[string]string{}
	for _, variable := range currentVariables {
		units[variable.Name] = variable.Unit
	}
	for _, variable := range dailyVariables {
		units[variable.Name] = variable.Unit
	}
	return units
}

// CurrentGrid is the map's grid of current values. Arrays hold one value per
// lattice node, row by row from the south-west: the value at index j × nx + i
// lies at lon0 + i × step, lat0 + j × step. Nodes outside the land mask are
// null.
type CurrentGrid struct {
	Source      string            `json:"source"`
	Model       string            `json:"model"`
	Attribution string            `json:"attribution"`
	ValidAt     time.Time         `json:"validAt"`
	FetchedAt   time.Time         `json:"fetchedAt"`
	Grid        GridSpec          `json:"grid"`
	Units       map[string]string `json:"units"`
	Values      map[string]Values `json:"values"`
	// ForecastFetchedAt is when the forecast giving uvIndex and
	// precipitationProbabilityPct was fetched; null without one.
	ForecastFetchedAt *time.Time `json:"forecastFetchedAt"`
}

// GridSpec describes the lattice of a grid response.
type GridSpec struct {
	ID   string  `json:"id"`
	Lon0 float64 `json:"lon0"`
	Lat0 float64 `json:"lat0"`
	Step float64 `json:"step"`
	Nx   int     `json:"nx"`
	Ny   int     `json:"ny"`
}

type encodedGrid struct {
	json, gzipped []byte
}

// BuildCurrentGrid assembles the current grid from a snapshot. Whether it is
// stale is left to the client, from validAt and the layer catalogue, since
// the encoded grid is cached until the next run.
func BuildCurrentGrid(snapshot *Snapshot) (*CurrentGrid, error) {
	current := snapshot.Current
	if current == nil {
		return nil, ErrNoData
	}
	size := snapshot.Lattice.Size()
	response := &CurrentGrid{
		Source: current.Source, Model: current.Model, Attribution: openmeteo.Attribution,
		ValidAt: current.SourceTime, FetchedAt: current.FetchedAt,
		Grid: GridSpec{ID: snapshot.GridID, Lon0: snapshot.Lattice.Lon0, Lat0: snapshot.Lattice.Lat0,
			Step: snapshot.Lattice.Step, Nx: snapshot.Lattice.Nx, Ny: snapshot.Lattice.Ny},
		Units:  Units(),
		Values: map[string]Values{},
	}
	if forecast := snapshot.Forecast; forecast != nil {
		fetched := forecast.FetchedAt
		response.ForecastFetchedAt = &fetched
	}
	for _, variable := range currentVariables {
		source := currentValues(snapshot, variable)
		if source == nil {
			continue
		}
		values := make(Values, size)
		for index := range values {
			values[index] = Value(source[index])
		}
		response.Values[variable.Name] = values
	}
	return response, nil
}

// currentValues returns a variable's current values over the lattice: from
// the current run, or from the forecast's hour nearest the current time.
func currentValues(snapshot *Snapshot, variable variable) []float32 {
	if variable.Current != "" {
		if field := snapshot.Current.Fields[variable.Current]; field != nil {
			return field.At(0)
		}
		return nil
	}
	if snapshot.Forecast == nil {
		return nil
	}
	field := snapshot.Forecast.Fields[openmeteo.HourlyPrefix+variable.Hourly]
	if field == nil {
		return nil
	}
	step, ok := field.StepAt(snapshot.Current.SourceTime)
	if !ok {
		return nil
	}
	return field.At(step)
}

// CurrentGridJSON returns the current grid encoded, and gzipped, once per
// snapshot.
func (service *Service) CurrentGridJSON(ctx context.Context) (plain, gzipped []byte, err error) {
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		return nil, nil, err
	}
	var buildErr error
	snapshot.once.Do(func() {
		response, err := BuildCurrentGrid(snapshot)
		if err != nil {
			buildErr = err
			return
		}
		body, err := json.Marshal(response)
		if err != nil {
			buildErr = err
			return
		}
		snapshot.encoded = &encodedGrid{json: body, gzipped: compress(body)}
	})
	if buildErr != nil {
		return nil, nil, buildErr
	}
	if snapshot.encoded == nil {
		return nil, nil, ErrNoData
	}
	return snapshot.encoded.json, snapshot.encoded.gzipped, nil
}

func compress(body []byte) []byte {
	var out bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&out, gzip.BestCompression)
	_, _ = writer.Write(body)
	_ = writer.Close()
	return out.Bytes()
}

// corners returns the four nodes around a point, or false outside the
// lattice.
func corners(lattice grid.Lattice, lon, lat float64) ([4]grid.Weight, bool) {
	return lattice.Corners(lon, lat)
}
