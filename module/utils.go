package module

import (
	"fmt"
	"reflect"
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
	case reflect.TypeOf((*model.HassEntity)(nil)).Elem():
		toks := strings.Split(data.(string), ".")
		if len(toks) != 2 {
			err = fmt.Errorf("Unable to parse entity %s", data.(string))
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
