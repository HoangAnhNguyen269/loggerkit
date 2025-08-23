package logger

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Level định nghĩa cấp độ log
type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
)

func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return "", fmt.Errorf("unknown level %q", s)
	}
}

// Text/JSON compatibility
func (l *Level) UnmarshalText(b []byte) error {
	v, err := ParseLevel(string(b))
	if err != nil {
		return err
	}
	*l = v
	return nil
}

func (l Level) MarshalText() ([]byte, error) {
	return []byte(string(l)), nil
}

func (l *Level) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return l.UnmarshalText([]byte(s))
}

func (l Level) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(l))
}
