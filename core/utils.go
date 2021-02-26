package core

import (
	"regexp"

	"github.com/nmaupu/gotomation/logging"
)

// StringInSliceP checks whether a string is present in the given slice (using regexp patterns)
func StringInSliceP(str string, patterns []string) bool {
	l := logging.NewLogger("StringInSliceP")
	for _, p := range patterns {

		re, err := regexp.Compile(p)
		if err != nil {
			l.Error().Err(err).Str("pattern", p).Msg("Pattern is not correct")
			continue
		}

		l.Trace().Str("candidate", str).Str("pattern", p).Bool("result", re.MatchString(str)).Msg("Checking pattern")

		if re.MatchString(str) {
			return true
		}
	}

	return false
}
