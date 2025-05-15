package config

import (
	"reflect"
)

// MergeStructs recursively merges non-zero fields from src into dst, including fields that are nil or missing in dst
func MergeStructs(dst, src any) {
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(src).Elem()

	// Only merge if both are structs
	if dstVal.Kind() != reflect.Struct || srcVal.Kind() != reflect.Struct {
		return
	}

	dstType := dstVal.Type()
	srcType := srcVal.Type()
	fieldNames := map[string]struct{}{}

	// Collect all field names from both structs
	for i := range dstVal.NumField() {
		fieldNames[dstType.Field(i).Name] = struct{}{}
	}
	for i := range srcVal.NumField() {
		fieldNames[srcType.Field(i).Name] = struct{}{}
	}

	for fieldName := range fieldNames {
		_, dstOk := dstType.FieldByName(fieldName)
		_, srcOk := srcType.FieldByName(fieldName)
		var dstFieldVal, srcFieldVal reflect.Value
		if dstOk {
			dstFieldVal = dstVal.FieldByName(fieldName)
		}
		if srcOk {
			srcFieldVal = srcVal.FieldByName(fieldName)
		}

		// If the field exists in src but not in dst, or dst is zero/nil, set it from src
		if srcOk && (!dstOk || (dstOk && isZeroValue(dstFieldVal))) {
			if dstOk && dstFieldVal.CanSet() && srcFieldVal.IsValid() && srcFieldVal.Type() == dstFieldVal.Type() {
				dstFieldVal.Set(srcFieldVal)
			}
			continue
		}

		// If the field exists in both, merge recursively or set if non-zero in src
		if dstOk && srcOk && dstFieldVal.CanSet() && srcFieldVal.IsValid() && dstFieldVal.Type() == srcFieldVal.Type() {
			switch dstFieldVal.Kind() {
			case reflect.Struct:
				MergeStructs(dstFieldVal.Addr().Interface(), srcFieldVal.Addr().Interface())
			case reflect.Ptr:
				if !srcFieldVal.IsNil() {
					if dstFieldVal.IsNil() {
						dstFieldVal.Set(reflect.New(dstFieldVal.Type().Elem()))
					}
					MergeStructs(dstFieldVal.Interface(), srcFieldVal.Interface())
				}
			case reflect.Slice, reflect.Array:
				if srcFieldVal.Len() > 0 {
					dstFieldVal.Set(srcFieldVal)
				}
			default:
				zero := reflect.Zero(dstFieldVal.Type()).Interface()
				if !reflect.DeepEqual(srcFieldVal.Interface(), zero) {
					dstFieldVal.Set(srcFieldVal)
				}
			}
		}
	}
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func:
		return v.IsNil()
	case reflect.Array:
		zero := true
		for i := range v.Len() {
			zero = zero && isZeroValue(v.Index(i))
		}
		return zero
	default:
		zero := reflect.Zero(v.Type()).Interface()
		return reflect.DeepEqual(v.Interface(), zero)
	}
}

// MergeWithDefaultConfig returns a config that combines user config with defaults
func MergeWithDefaultConfig(userConfig *UserConfig) Config {
	mergedConfig := GetConfig()
	if userConfig != nil {
		MergeStructs(&mergedConfig, userConfig)
	}
	return mergedConfig
}
