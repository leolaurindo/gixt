package indexdesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

func Load(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return map[string]string{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read descriptions: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse descriptions: %w", err)
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

func Save(path string, values map[string]string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("description override path is empty")
	}
	buf, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return fmt.Errorf("encode descriptions: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write descriptions: %w", err)
	}
	return nil
}

func Sorted(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for k := range values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func Normalize(desc string) string {
	return strings.TrimSpace(desc)
}
