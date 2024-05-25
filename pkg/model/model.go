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

func (e Exif) GetRawCoordinates() (string, bool) {
	raw, ok := e.Data["GeolocationPosition"]
	if !ok {
		return "", false
	}

	content, ok := raw.(string)
	return content, ok
}
