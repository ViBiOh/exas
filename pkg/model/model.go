package model

import "time"

// Geocode data
type Geocode struct {
	Address   map[string]string `json:"address,omitempty"`
	Latitude  float64           `json:"lat,omitempty"`
	Longitude float64           `json:"lon,omitempty"`
}

// HasAddress checks if struct has address
func (g Geocode) HasAddress() bool {
	return len(g.Address) != 0
}

// HasCoordinates checks if struct has coordinates
func (g Geocode) HasCoordinates() bool {
	return g.Latitude != 0 && g.Longitude != 0
}

// Exif data
type Exif struct {
	Date        time.Time      `json:"date,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Description string         `json:"description,omitempty"`
	Geocode     Geocode        `json:"geocode,omitempty"`
}

// IsZero checks if struct is empty or not
func (e Exif) IsZero() bool {
	return len(e.Data) == 0 && len(e.Description) == 0
}
