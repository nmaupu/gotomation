package core

import (
	"testing"
	"time"
)

func TestSchedulesDays_AsFlag(t *testing.T) {
	tests := []struct {
		name string
		s    SchedulesDays
		want int
	}{
		{
			name: "test_all",
			s:    SchedulesDays("monday, tuesday,wednesday, thursday, friday,   saturday  , sunday"),
			want: 0b1111111,
		},
		{
			name: "test_monday",
			s:    SchedulesDays("monday"),
			want: 0b0000010,
		},
		{
			name: "test_tuesday",
			s:    SchedulesDays("tuesday"),
			want: 0b0000100,
		},
		{
			name: "test_wednesday",
			s:    SchedulesDays("wednesday"),
			want: 0b0001000,
		},
		{
			name: "test_thursday",
			s:    SchedulesDays("thursday"),
			want: 0b0010000,
		},
		{
			name: "test_friday",
			s:    SchedulesDays("friday"),
			want: 0b0100000,
		},
		{
			name: "test_saturday",
			s:    SchedulesDays("saturday"),
			want: 0b1000000,
		},
		{
			name: "test_sunday",
			s:    SchedulesDays("sunday"),
			want: 0b0000001,
		},
		{
			name: "test_week",
			s:    SchedulesDays("week"),
			want: 0b0111110,
		},
		{
			name: "test_weekEnd",
			s:    SchedulesDays("weekEnd"),
			want: 0b1000001,
		},
		{
			name: "test_week_sat_sun",
			s:    SchedulesDays("week,saturday,sunday"),
			want: 0b1111111,
		},
		{
			name: "test_weekEnd_monday",
			s:    SchedulesDays("weekEnd,monday"),
			want: 0b1000011,
		},
		{
			name: "test_week_sunday",
			s:    SchedulesDays("week,sunday"),
			want: 0b0111111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.AsFlag(); got != tt.want {
				t.Errorf("SchedulesDays.AsFlag() = %07b, want %07b", got, tt.want)
			}
		})
	}
}

func TestSchedulesDays_IsScheduled(t *testing.T) {
	monday := time.Date(2021, time.February, int(time.Monday), 0, 0, 0, 0, time.Local)
	tuesday := time.Date(2021, time.February, int(time.Tuesday), 0, 0, 0, 0, time.Local)
	wednesday := time.Date(2021, time.February, int(time.Wednesday), 0, 0, 0, 0, time.Local)
	thursday := time.Date(2021, time.February, int(time.Thursday), 0, 0, 0, 0, time.Local)
	friday := time.Date(2021, time.February, int(time.Friday), 0, 0, 0, 0, time.Local)
	saturday := time.Date(2021, time.February, int(time.Saturday), 0, 0, 0, 0, time.Local)
	sunday := time.Date(2021, time.February, int(time.Sunday), 0, 0, 0, 0, time.Local)
	tests := []struct {
		name string
		s    SchedulesDays
		t    time.Time
		want bool
	}{
		{
			name: "test_monday_week",
			s:    SchedulesDays("week"),
			t:    monday,
			want: true,
		},
		{
			name: "test_tuesday_week",
			s:    SchedulesDays("week"),
			t:    tuesday,
			want: true,
		},
		{
			name: "test_wednesday_week",
			s:    SchedulesDays("week"),
			t:    wednesday,
			want: true,
		},
		{
			name: "test_thursday_week",
			s:    SchedulesDays("week"),
			t:    thursday,
			want: true,
		},
		{
			name: "test_friday_week",
			s:    SchedulesDays("week"),
			t:    friday,
			want: true,
		},
		{
			name: "test_saturday_week",
			s:    SchedulesDays("week"),
			t:    saturday,
			want: false,
		},
		{
			name: "test_sunday_week",
			s:    SchedulesDays("week"),
			t:    sunday,
			want: false,
		},
		{
			name: "test_monday_weekEnd",
			s:    SchedulesDays("weekEnd"),
			t:    monday,
			want: false,
		},
		{
			name: "test_friday_weekEnd",
			s:    SchedulesDays("weekEnd"),
			t:    friday,
			want: false,
		},
		{
			name: "test_saturday_weekEnd",
			s:    SchedulesDays("weekEnd"),
			t:    saturday,
			want: true,
		},
		{
			name: "test_sunday_weekEnd",
			s:    SchedulesDays("weekEnd"),
			t:    sunday,
			want: true,
		},
		{
			name: "test_even_days_monday",
			s:    SchedulesDays("sunday,tuesday,thursday,saturday"),
			t:    monday,
			want: false,
		},
		{
			name: "test_even_days_tuesday",
			s:    SchedulesDays("sunday,tuesday,thursday,saturday"),
			t:    tuesday,
			want: true,
		},
		{
			name: "test_even_days_wednesday",
			s:    SchedulesDays("sunday,tuesday,thursday,saturday"),
			t:    wednesday,
			want: false,
		},
		{
			name: "test_even_days_thursday",
			s:    SchedulesDays("sunday,tuesday,thursday,saturday"),
			t:    thursday,
			want: true,
		},
		{
			name: "test_even_days_friday",
			s:    SchedulesDays("sunday,tuesday,thursday,saturday"),
			t:    friday,
			want: false,
		},
		{
			name: "test_even_days_saturday",
			s:    SchedulesDays("sunday,tuesday,thursday,saturday"),
			t:    saturday,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsScheduled(tt.t); got != tt.want {
				t.Errorf("SchedulesDays.IsScheduled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaterSchedules_Sort(t *testing.T) {
	wantedSchedules := map[SchedulesDays][]HeaterSchedule{
		"week": {
			{
				Beg: time.Date(0, 0, 0, 8, 0, 0, 0, time.Local),
				End: time.Date(0, 0, 0, 9, 0, 0, 0, time.Local),
			},
			{
				Beg: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
				End: time.Date(0, 0, 0, 11, 0, 0, 0, time.Local),
			},
			{
				Beg: time.Date(0, 0, 0, 12, 0, 0, 0, time.Local),
				End: time.Date(0, 0, 0, 13, 0, 0, 0, time.Local),
			},
			{
				Beg: time.Date(0, 0, 0, 14, 0, 0, 0, time.Local),
				End: time.Date(0, 0, 0, 15, 0, 0, 0, time.Local),
			},
		},
	}

	tests := []struct {
		name      string
		schedules map[SchedulesDays][]HeaterSchedule
		want      map[SchedulesDays][]HeaterSchedule
	}{
		{
			name: "sort_test_1",
			schedules: map[SchedulesDays][]HeaterSchedule{
				"week": {
					{
						Beg: time.Date(0, 0, 0, 12, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 13, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 11, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 14, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 15, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 8, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 9, 0, 0, 0, time.Local),
					},
				},
			},
			want: wantedSchedules,
		},
		{
			name: "sort_test_2",
			schedules: map[SchedulesDays][]HeaterSchedule{
				"week": {
					{
						Beg: time.Date(0, 0, 0, 14, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 15, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 12, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 13, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 11, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 8, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 9, 0, 0, 0, time.Local),
					},
				},
			},
			want: wantedSchedules,
		},
		{
			name: "sort_test_3",
			schedules: map[SchedulesDays][]HeaterSchedule{
				"week": {
					{
						Beg: time.Date(0, 0, 0, 8, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 9, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 11, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 12, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 13, 0, 0, 0, time.Local),
					},
					{
						Beg: time.Date(0, 0, 0, 14, 0, 0, 0, time.Local),
						End: time.Date(0, 0, 0, 15, 0, 0, 0, time.Local),
					},
				},
			},
			want: wantedSchedules,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &HeaterSchedules{
				Scheds: tt.schedules,
			}
			c.Sort()

			for key, sched := range c.Scheds {
				sortedSlice := sched
				wantedSlice := tt.want[key]
				for k, v := range sortedSlice {
					if !v.Beg.Equal(wantedSlice[k].Beg) || !v.End.Equal(wantedSlice[k].End) {
						t.Errorf("Sort is not ok, sorted=[%v,%v] wanted=[%v,%v]", v.Beg, v.End, wantedSlice[k].Beg, wantedSlice[k].End)
					}
				}
			}
		})
	}
}

func TestHeaterSchedules_GetTemperatureToSet(t *testing.T) {
	scheds := map[SchedulesDays][]HeaterSchedule{
		"week,weekEnd": {
			{
				Beg:     time.Date(0, 0, 0, 14, 0, 0, 0, time.Local),
				End:     time.Date(0, 0, 0, 15, 0, 0, 0, time.Local),
				Eco:     14,
				Comfort: 15,
			},
			{
				Beg:     time.Date(0, 0, 0, 12, 0, 0, 0, time.Local),
				End:     time.Date(0, 0, 0, 13, 0, 0, 0, time.Local),
				Eco:     12,
				Comfort: 13,
			},
			{
				Beg:     time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
				End:     time.Date(0, 0, 0, 11, 0, 0, 0, time.Local),
				Eco:     10,
				Comfort: 11,
			},
			{
				Beg:     time.Date(0, 0, 0, 8, 0, 0, 0, time.Local),
				End:     time.Date(0, 0, 0, 9, 0, 0, 0, time.Local),
				Eco:     8,
				Comfort: 9,
			},
		},
	}
	type fields struct {
		Scheds     map[SchedulesDays][]HeaterSchedule
		DefaultEco float64
	}
	tests := []struct {
		name   string
		fields fields
		t      time.Time
		want   float64
	}{
		{
			name: "test_1",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 6, 0, 0, 0, time.Local),
			want: 16,
		},
		{
			name: "test_2",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 8, 30, 0, 0, time.Local),
			want: 9,
		},
		{
			name: "test_3",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 9, 30, 0, 0, time.Local),
			want: 8,
		},
		{
			name: "test_4",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 10, 10, 0, 0, time.Local),
			want: 11,
		},
		{
			name: "test_5",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 11, 30, 0, 0, time.Local),
			want: 10,
		},
		{
			name: "test_6",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 12, 30, 0, 0, time.Local),
			want: 13,
		},
		{
			name: "test_7",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 13, 30, 0, 0, time.Local),
			want: 12,
		},
		{
			name: "test_8",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 14, 30, 0, 0, time.Local),
			want: 15,
		},
		{
			name: "test_9",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 15, 30, 0, 0, time.Local),
			want: 14,
		},
		{
			name: "test_10",
			fields: fields{
				Scheds:     scheds,
				DefaultEco: 16,
			},
			t:    time.Date(2021, 02, 28, 21, 00, 0, 0, time.Local),
			want: 14,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &HeaterSchedules{
				Scheds:     tt.fields.Scheds,
				DefaultEco: tt.fields.DefaultEco,
			}
			if got := c.GetTemperatureToSet(tt.t); got != tt.want {
				t.Errorf("HeaterSchedules.GetTemperatureToSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
