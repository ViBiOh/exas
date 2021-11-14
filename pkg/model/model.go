package model

import "time"

// Geocode data
type Geocode struct {
	Address   map[string]string `json:"address"`
	Latitude  string            `json:"lat"`
	Longitude string            `json:"lon"`
}

// IsZero checks if struct is empty or not
func (g Geocode) IsZero() bool {
	return len(g.Address) == 0
}

// Exif data
type Exif struct {
	Date    time.Time              `json:"date"`
	Geocode Geocode                `json:"geocode"`
	Data    map[string]interface{} `json:"data"`
}

// IsZero checks if struct is empty or not
func (e Exif) IsZero() bool {
	return len(e.Data) == 0
}
