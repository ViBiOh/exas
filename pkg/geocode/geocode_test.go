package geocode

import (
	"errors"
	"math"
	"strings"
	"testing"
)

const float64EqualityThreshold = 0.000001

func TestConvertDegreeMinuteSecondToDecimal(t *testing.T) {
	type args struct {
		location string
	}

	cases := []struct {
		intention string
		args      args
		want      float64
		wantErr   error
	}{
		{
			"empty",
			args{
				location: "",
			},
			0,
			errors.New("unable to parse GPS data"),
		},
		{
			"north",
			args{
				location: "1 deg 2' 3\" N",
			},
			1.034167,
			nil,
		},
		{
			"west",
			args{
				location: "1 deg 2' 3\" W",
			},
			-1.034167,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			got, gotErr := convertDegreeMinuteSecondToDecimal(tc.args.location)

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			} else if math.Abs(got-tc.want) > float64EqualityThreshold {
				failed = true
			}

			if failed {
				t.Errorf("convertDegreeMinuteSecondToDecimal() = (%f, `%s`), want (%f, `%s`)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}
