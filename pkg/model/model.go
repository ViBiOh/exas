package model

import "time"

type Exif struct {
	Date    time.Time      `json:"date,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Geocode Geocode        `json:"geocode,omitempty"`
}

func (e Exif) IsZero() bool {
	return !e.HasData()
}

func (e Exif) HasData() bool {
	return len(e.Data) != 0
}

type Geocode struct {
	Address   map[string]string `json:"address,omitempty"`
	Latitude  float64           `json:"lat,omitempty"`
	Longitude float64           `json:"lon,omitempty"`
}

func (g Geocode) HasAddress() bool {
	return len(g.Address) != 0
}

func (g Geocode) HasCoordinates() bool {
	return g.Latitude != 0 && g.Longitude != 0
}
