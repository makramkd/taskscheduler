package agent

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

func parseFrequency(frequency string) (time.Duration, error) {
	split := strings.Split(frequency, " ")
	if len(split) != 3 {
		return 0, errors.New("frequency must be of the form 'every <number> (seconds|minutes|hours)")
	}
	// only "every" is supported
	if split[0] != "every" {
		return 0, errors.New("frequency must be of the form 'every <number> (seconds|minutes|hours)")
	}

	dur, err := strconv.Atoi(split[1])
	if err != nil {
		return 0, errors.New("frequency must be of the form 'every <number> (seconds|minutes|hours)")
	}

	if split[2] != "seconds" && split[2] != "minutes" && split[2] != "hours" {
		return 0, errors.New("frequency must be of the form 'every <number> (seconds|minutes|hours)")
	}

	switch split[2] {
	case "seconds":
		return time.Duration(dur) * time.Second, nil
	case "minutes":
		return time.Duration(dur) * time.Minute, nil
	case "hours":
		return time.Duration(dur) * time.Hour, nil
	default:
		// impossible case
		return time.Duration(0), errors.New("bad error")
	}
}
