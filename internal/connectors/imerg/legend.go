package imerg

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// ColormapURL is the colour map GIBS renders the layer with; its "Rain Rate"
// entries give the legend's colours. A test checks LegendStops against a
// recorded excerpt of it.
const ColormapURL = "https://gibs.earthdata.nasa.gov/colormaps/v1.3/GPM_Precipitation_Rate.xml"

// LegendStop is a rate in mm/h and the colour GIBS draws it in.
type LegendStop struct {
	RateMmH float64 `json:"rateMmH"`
	Color   string  `json:"color"`
}

// LegendStops are the colours GIBS draws at round rates. Rates under 0.1 mm/h
// are transparent.
var LegendStops = []LegendStop{
	{RateMmH: 0.1, Color: "#00764e"},
	{RateMmH: 0.5, Color: "#45c000"},
	{RateMmH: 1, Color: "#c3e400"},
	{RateMmH: 2, Color: "#ffb006"},
	{RateMmH: 5, Color: "#ff4730"},
	{RateMmH: 10, Color: "#e70000"},
	{RateMmH: 20, Color: "#9c0000"},
	{RateMmH: 50, Color: "#3f0000"},
}

// ColormapEntry is one class of a colour map: rates from From up to To.
type ColormapEntry struct {
	From, To float64
	Color    string
}

type wireColorMaps struct {
	Maps []struct {
		Title   string `xml:"title,attr"`
		Entries []struct {
			RGB         string `xml:"rgb,attr"`
			Transparent bool   `xml:"transparent,attr"`
			Value       string `xml:"sourceValue,attr"`
		} `xml:"Entries>ColorMapEntry"`
	} `xml:"ColorMap"`
}

// DecodeColormap reads the "Rain Rate" classes of a GIBS colour map.
func DecodeColormap(payload []byte) ([]ColormapEntry, error) {
	var wire wireColorMaps
	if err := xml.Unmarshal(payload, &wire); err != nil {
		return nil, fmt.Errorf("decode colour map: %w", err)
	}
	for _, colorMap := range wire.Maps {
		if colorMap.Title != "Rain Rate" {
			continue
		}
		entries := make([]ColormapEntry, 0, len(colorMap.Entries))
		for _, entry := range colorMap.Entries {
			var red, green, blue int
			if _, err := fmt.Sscanf(entry.RGB, "%d,%d,%d", &red, &green, &blue); err != nil {
				return nil, fmt.Errorf("colour %q: %w", entry.RGB, err)
			}
			bounds := strings.Split(strings.Trim(entry.Value, "[]()"), ",")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("value range %q", entry.Value)
			}
			from, err := strconv.ParseFloat(bounds[0], 64)
			if err != nil {
				return nil, fmt.Errorf("value range %q: %w", entry.Value, err)
			}
			to, err := strconv.ParseFloat(strings.TrimPrefix(bounds[1], "+"), 64)
			if err != nil {
				return nil, fmt.Errorf("value range %q: %w", entry.Value, err)
			}
			entries = append(entries, ColormapEntry{From: from, To: to, Color: fmt.Sprintf("#%02x%02x%02x", red, green, blue)})
		}
		return entries, nil
	}
	return nil, fmt.Errorf("colour map has no Rain Rate classes")
}
