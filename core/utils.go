package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/nmaupu/gotomation/model"
)

// MapstructureDecodeHook is used to decode a string to time.Duration
func MapstructureDecodeHook(from, to reflect.Type, data interface{}) (interface{}, error) {
	if from.Kind() != reflect.String {
		return data, nil
	}

	var result interface{}
	var err error

	switch to {
	case reflect.TypeOf((*time.Duration)(nil)).Elem():
		result, err = time.ParseDuration(data.(string))
	case reflect.TypeOf((*time.Time)(nil)).Elem():
		toks := strings.Split(data.(string), ":")
		if len(toks) != 3 {
			return nil, fmt.Errorf("Unable to parse time %s", data.(string))
		}

		hour, errHour := strconv.Atoi(toks[0])
		minute, errMinute := strconv.Atoi(toks[1])
		second, errSecond := strconv.Atoi(toks[2])
		if errHour != nil || errMinute != nil || errSecond != nil {
			return nil, fmt.Errorf("Unable to parse time %s", data.(string))
		}

		now := time.Now()
		result = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, time.Local)
	case reflect.TypeOf((*model.HassEntity)(nil)).Elem():
		toks := strings.Split(data.(string), ".")
		if len(toks) != 2 {
			return nil, fmt.Errorf("Unable to parse entity %s", data.(string))
		}
		result = model.HassEntity{
			Domain:   toks[0],
			EntityID: toks[1],
		}
	default:
		return data, nil
	}

	return result, err
}

// StringInSlice checks whether a string is present in the given slice
func StringInSlice(str string, s []string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
