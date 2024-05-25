package model

import (
	"time"
)

type Exif struct {
	Date        time.Time      `json:"date,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Coordinates *LatLng        `json:"coordinates,omitempty"`
}

func (e Exif) IsZero() bool {
	return !e.HasData()
}

func (e Exif) HasData() bool {
	return len(e.Data) != 0
}

func (e Exif) HasAddress() bool {
	_, ok := e.Data["GeolocationCity"]
	return ok
}

func (e Exif) GetRawCoordinates() string {
	return e.GetString("GeolocationPosition")
}

func (e Exif) GetCity() string {
	return e.GetString("GeolocationCity")
}

func (e Exif) GetCountry() string {
	return e.GetString("GeolocationCountry")
}

func (e Exif) GetString(key string) string {
	raw, ok := e.Data[key]
	if !ok {
		return ""
	}

	content, _ := raw.(string)
	return content
}
