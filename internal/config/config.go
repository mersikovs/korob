package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	ErrConfigPathEmpty = errors.New("config path is set but empty")
	ErrConfigFlagNoArg = errors.New("config flag requires an argument")
)

func loadJSONConfig(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode config %q: %w", path, err)
	}
	return nil
}

func scanConfigFlag(args, aliases []string) (path string, set bool, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		for _, alias := range aliases {
			if a == alias {
				if i+1 >= len(args) {
					return "", false, fmt.Errorf("%w: %s", ErrConfigFlagNoArg, a)
				}
				path = args[i+1]
				set = true
				i++
				break
			}
			if strings.HasPrefix(a, alias+"=") {
				path = strings.TrimPrefix(a, alias+"=")
				set = true
				break
			}
		}
	}
	if set && path == "" {
		return "", true, ErrConfigPathEmpty
	}
	return path, set, nil
}

func lookupConfigEnv(env EnvSource, name string) (path string, set bool, err error) {
	if name == "" {
		return "", false, nil
	}
	v, ok := env.LookupEnv(name)
	if !ok {
		return "", false, nil
	}
	if v == "" {
		return "", true, fmt.Errorf("%s: %w", name, ErrConfigPathEmpty)
	}
	return v, true, nil
}

func lookupNonEmptyEnv(env EnvSource, key string) (*string, error) {
	v, ok := env.LookupEnv(key)
	if !ok {
		return nil, nil
	}
	if v == "" {
		return nil, fmt.Errorf("env %s is set but empty", key)
	}
	return &v, nil
}

func applyIfSet(dst *string, src *string) {
	if src != nil {
		*dst = *src
	}
}

func applyIfSetPtr[T any](dst **T, src *T) {
	if src != nil {
		*dst = src
	}
}

func positiveIntFlag(name string, target *int) func(string) error {
	return func(s string) error {
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("flag -%s: not a number %q: %w", name, s, err)
		}
		if v <= 0 {
			return fmt.Errorf("flag -%s: must be a positive integer, got %d", name, v)
		}
		*target = v
		return nil
	}
}

func getPositiveIntEnv(key, value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil {
		return v, fmt.Errorf("env %s: not a number %q: %w", key, value, err)
	}
	if v <= 0 {
		return v, fmt.Errorf("env %s: must be a positive integer, got %d", key, v)
	}
	return v, nil
}
