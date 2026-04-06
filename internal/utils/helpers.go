package utils

import "strconv"

func GetString(m map[string]any, key string) *string {
	v, ok := m[key].(string)
	if !ok {
		return nil
	}
	return &v
}

func GetFloat(m map[string]any, key string) *float64 {
	v, ok := m[key].(float64)
	if !ok {
		return nil
	}
	return &v
}

func GetInt(m map[string]any, key string) *int {
	v, ok := m[key].(float64)
	if !ok {
		return nil
	}
	final := int(v)
	return &final
}

func GetFloatFromString(m map[string]any, key string) *float64 {
	v, ok := m[key].(string)
	if !ok {
		return nil
	}
	final, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &final
}

func GetIntFromString(m map[string]any, key string) *int {
	v, ok := m[key].(string)
	if !ok {
		return nil
	}
	final, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &final
}
