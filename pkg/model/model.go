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
	Data    map[string]interface{} `json:"data"`
	Date    time.Time              `json:"date"`
	Geocode Geocode                `json:"geocode"`
}

// IsZero checks if struct is empty or not
func (e Exif) IsZero() bool {
	return len(e.Data) == 0
}

// StorageItem describe item on a storage provider
type StorageItem struct {
	Date     time.Time `json:"date"`
	Name     string    `json:"name"`
	Pathname string    `json:"pathname"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
}
