package model

import "testing"

func TestJoinEntities(t *testing.T) {
	entities := []HassEntity{
		{
			Domain:   "foo1",
			EntityID: "bar1",
		},
		{
			Domain:   "foo2",
			EntityID: "bar2",
		},
	}
	tests := []struct {
		name     string
		elems    []HassEntity
		sep      string
		toDelete []string
		want     string
	}{
		{
			name:  "test 1",
			elems: entities,
			sep:   ", ",
			want:  "foo1.bar1, foo2.bar2",
		},
		{
			name:  "test 2",
			elems: []HassEntity{},
			sep:   ", ",
			want:  "",
		},
		{
			name: "test 3",
			elems: []HassEntity{
				entities[0],
			},
			sep:  ", ",
			want: "foo1.bar1",
		},
		{
			name: "test 3",
			elems: []HassEntity{
				{
					Domain:   "sensor",
					EntityID: "hum_temp_last_seen",
				},
			},
			sep:      ", ",
			toDelete: []string{"_last_seen"},
			want:     "sensor.hum_temp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinEntities(tt.elems, tt.sep, tt.toDelete...); got != tt.want {
				t.Errorf("JoinEntities() = %v, want %v", got, tt.want)
			}
		})
	}
}
