package configure

import (
	"errors"
	"io/ioutil"
	"reflect"

	yaml "gopkg.in/yaml.v2"
)

var ErrConfigureObjectIsNotStruct = errors.New("configure object isn't struct")
var ErrRequiredFieldIsEmpty = errors.New("required filed hasn't been set")

func Load(config interface{}, file string) error {
	if err := processFile(config, file); err != nil {
		return err
	}
	return processTags(config)
}

func processFile(config interface{}, file string) error {
	if data, err := ioutil.ReadFile(file); err != nil {
		return err
	} else {
		return yaml.Unmarshal(data, config)
	}
}

func processTags(config interface{}) error {
	value := reflect.Indirect(reflect.ValueOf(config))
	if value.Kind() != reflect.Struct {
		return ErrConfigureObjectIsNotStruct
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		fieldValue := value.Field(i)

		for fieldValue.Kind() == reflect.Ptr {
			fieldValue = fieldValue.Elem()
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			if err := processTags(fieldValue.Addr().Interface()); err != nil {
				return err
			}
		case reflect.Slice:
			for i := 0; i < fieldValue.Len(); i++ {
				if reflect.Indirect(fieldValue.Index(i)).Kind() == reflect.Struct {
					if err := processTags(fieldValue.Index(i).Addr().Interface()); err != nil {
						return err
					}
				}
			}
		default:
			if isBlank := reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(fieldValue.Type()).Interface()); isBlank {
				if value := fieldType.Tag.Get("default"); value != "" {
					if err := yaml.Unmarshal([]byte(value), fieldValue.Addr().Interface()); err != nil {
						return err
					}
				} else if fieldType.Tag.Get("required") == "true" {
					return ErrRequiredFieldIsEmpty
				}
			}
		}
	}
	return nil
}
