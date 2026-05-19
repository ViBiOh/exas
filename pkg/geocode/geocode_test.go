package geocode

import (
	"errors"
	"math"
	"strings"
	"testing"
)

const float64EqualityThreshold = 0.000001

func TestConvertDegreeMinuteSecondToDecimal(t *testing.T) {
	t.Parallel()

	type args struct {
		location string
	}

	cases := map[string]struct {
		args    args
		want    float64
		wantErr error
	}{
		"empty": {
			args{
				location: "",
			},
			0,
			errors.New("parse GPS data"),
		},
		"north": {
			args{
				location: "1 deg 2' 3\" N",
			},
			1.034167,
			nil,
		},
		"west": {
			args{
				location: "1 deg 2' 3\" W",
			},
			-1.034167,
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			got, gotErr := convertDegreeMinuteSecondToDecimal(tc.args.location)

			switch {
			case tc.wantErr == nil && gotErr != nil:
				t.Errorf("unexpected error: %s", gotErr)
			case tc.wantErr != nil && gotErr == nil:
				t.Errorf("expected error containing %q, got nil", tc.wantErr)
			case tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()):
				t.Errorf("error = %q, want containing %q", gotErr, tc.wantErr)
			}

			if math.Abs(got-tc.want) > float64EqualityThreshold {
				t.Errorf("result = %f, want %f", got, tc.want)
			}
		})
	}
}
