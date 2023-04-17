package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a JSON (un)marshallable version of time.Duration.
type Duration time.Duration

// MarshalJSON implements the json.Marshaler interface.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("error parsing duration: %w", err)
		}
		*d = Duration(tmp)
		return nil
	default:
		return fmt.Errorf("invalid duration: %s", b)
	}
}
