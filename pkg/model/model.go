package model

import "time"

// Geocode data
type Geocode struct {
	Address   map[string]string `json:"address,omitempty"`
	Latitude  float64           `json:"lat,omitempty"`
	Longitude float64           `json:"lon,omitempty"`
}

// IsZero checks if struct is empty or not
func (g Geocode) IsZero() bool {
	return len(g.Address) == 0
}

// Exif data
type Exif struct {
	Data        map[string]interface{} `json:"data,omitempty"`
	Date        time.Time              `json:"date,omitempty"`
	Geocode     Geocode                `json:"geocode,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// IsZero checks if struct is empty or not
func (e Exif) IsZero() bool {
	return len(e.Data) == 0 && len(e.Description) == 0
}
