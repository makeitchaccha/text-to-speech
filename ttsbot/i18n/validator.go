package i18n

import (
	"fmt"
	"reflect"
)

func validateResource(s interface{}, path string) []error {
	var errs []error
	v := reflect.ValueOf(s)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		currentPath := fmt.Sprintf("%s.%s", path, fieldType.Name)

		switch fieldValue.Kind() {
		case reflect.String:
			if fieldValue.String() == "" {
				errs = append(errs, fmt.Errorf("field %s is an empty string", currentPath))
			}
		case reflect.Struct:
			nestedErrs := validateResource(fieldValue.Interface(), currentPath)
			errs = append(errs, nestedErrs...)
		case reflect.Ptr:
			if !fieldValue.IsNil() && fieldValue.Elem().Kind() == reflect.Struct {
				nestedErrs := validateResource(fieldValue.Interface(), currentPath)
				errs = append(errs, nestedErrs...)
			}
		}
	}

	return errs
}
