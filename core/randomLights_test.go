package core

import (
	"testing"
	"time"
)

func Test_randomLightsRoutine_CheckTime(t *testing.T) {
	type fields struct {
	}

	tests := []struct {
		name      string
		now       time.Time
		sunrise   time.Time
		startTime time.Time
		endTime   time.Time
		want      bool
	}{
		{
			name:      "test_1",
			now:       time.Date(2021, 07, 10, 17, 0, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 10, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 22, 0, 0, 0, time.Local),
			want:      false,
		},
		{
			name:      "test_2",
			now:       time.Date(2021, 07, 10, 19, 30, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 10, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 22, 0, 0, 0, time.Local),
			want:      true,
		},
		{
			name:      "test_3",
			now:       time.Date(2021, 07, 10, 23, 0, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 10, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 22, 0, 0, 0, time.Local),
			want:      false,
		},
		{
			name:      "test_4",
			now:       time.Date(2021, 07, 10, 17, 0, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 10, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 0, 45, 0, 0, time.Local),
			want:      false,
		},
		{
			name:      "test_5",
			now:       time.Date(2021, 07, 10, 19, 30, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 10, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 0, 45, 0, 0, time.Local),
			want:      true,
		},
		{
			name:      "test_6",
			now:       time.Date(2021, 07, 11, 0, 30, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 11, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 0, 45, 0, 0, time.Local),
			want:      true,
		},
		{
			name:      "test_7",
			now:       time.Date(2021, 07, 11, 2, 30, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 11, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 0, 45, 0, 0, time.Local),
			want:      false,
		},
		{
			name:      "test_8",
			now:       time.Date(2021, 07, 11, 8, 30, 0, 0, time.Local),
			sunrise:   time.Date(2021, 07, 11, 6, 0, 0, 0, time.Local),
			startTime: time.Date(0, 0, 0, 19, 0, 0, 0, time.Local),
			endTime:   time.Date(0, 0, 0, 0, 45, 0, 0, time.Local),
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &randomLightsRoutine{
				startTime: tt.startTime,
				endTime:   tt.endTime,
			}
			if got := r.CheckTime(tt.now, tt.sunrise); got != tt.want {
				t.Errorf("CheckTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
