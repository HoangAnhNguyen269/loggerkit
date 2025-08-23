package logger

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

func ParseEnv(s string) (Env, error) {
	switch strings.ToLower(s) {
	case "dev":
		return EnvDev, nil
	case "prod":
		return EnvProd, nil
	default:
		return "", fmt.Errorf("unknown env %q", s)
	}
}

// Text/JSON compatibility
func (e *Env) UnmarshalText(b []byte) error {
	v, err := ParseEnv(string(b))
	if err != nil {
		return err
	}
	*e = v
	return nil
}

func (e Env) MarshalText() ([]byte, error) {
	return []byte(string(e)), nil
}

func (e *Env) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return e.UnmarshalText([]byte(s))
}

func (e Env) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(e))
}
