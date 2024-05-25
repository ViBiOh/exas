package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type LatLng [2]float64

var ErrLatLngNeedsExactlyTwoValues = errors.New("LatLng needs exactly two values")

func ParseLatLng(raw string) (LatLng, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return LatLng{}, fmt.Errorf("`%s`: %w", raw, ErrLatLngNeedsExactlyTwoValues)
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return LatLng{}, fmt.Errorf("latitude is not a float `%s`", raw)
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return LatLng{}, fmt.Errorf("longitude is not a float `%s`", raw)
	}

	return LatLng{lat, lng}, nil
}
