package charts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"

	"github.com/zdnscloud/cement/slice"
)

const (
	TypeInt    = "int"
	TypeString = "string"
	TypeEnum   = "enum"

	configPath = "config/config.json"
)

type ChartConfig struct {
	Label       string   `json:"label"`
	JsonKey     string   `json:"jsonKey"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	ValidValues []string `json:"validValues,omitempty"`
	Min         int      `json:"min,omitempty"`
	Max         int      `json:"max,omitempty"`
	MinLen      int      `json:"minLen,omitempty"`
	MaxLen      int      `json:"maxLen,omitempty"`
}

type ChartConfigs []ChartConfig

func CheckConfigs(chartVersionDir string, configs map[string]interface{}) error {
	chartConfigs, err := LoadChartConfigs(chartVersionDir)
	if err != nil {
		return err
	}

	return chartConfigs.checkCriteria(configs)
}

func LoadChartConfigs(chartVersionDir string) (ChartConfigs, error) {
	content, err := ioutil.ReadFile(path.Join(chartVersionDir, configPath))
	if err != nil {
		return nil, fmt.Errorf("read chart config.json failed: %s", err.Error())
	}

	var configs ChartConfigs
	if err := json.Unmarshal(content, &configs); err != nil {
		return nil, fmt.Errorf("unmarshal chart config.json failed: %s", err.Error())
	}

	if err := configs.validate(); err != nil {
		return nil, fmt.Errorf("chart config.json invalid: %s", err.Error())
	}

	return configs, nil
}

func (cs ChartConfigs) validate() (err error) {
	for _, c := range cs {
		switch c.Type {
		case TypeInt:
			err = c.validateInt()
		case TypeString:
			err = c.validateStr()
		case TypeEnum:
			err = c.validateEnum()
		}

		if err != nil {
			return
		}
	}

	return
}

func (c *ChartConfig) validateInt() error {
	if len(c.ValidValues) > 0 || c.MinLen != 0 || c.MaxLen != 0 {
		return fmt.Errorf("integer field %s doesn't support validValues and minLen and maxLen", c.Label)
	}

	if c.Min == 0 && c.Max == 0 {
		return nil
	}

	if c.Min == 0 || c.Max == 0 {
		return fmt.Errorf("field %s min and max must set together", c.Label)
	}

	if c.Min >= c.Max {
		return fmt.Errorf("field %s min value shoud smaller than max", c.Label)
	}

	return nil
}

func (c *ChartConfig) validateStr() error {
	if len(c.ValidValues) > 0 || c.Min != 0 || c.Max != 0 {
		return fmt.Errorf("string field %s doesn't support validValues and min and max", c.Label)
	}

	if c.MinLen == 0 && c.MaxLen == 0 {
		return nil
	}

	if c.MinLen == 0 || c.MaxLen == 0 {
		return fmt.Errorf("field %s minLen and maxLen must set together", c.Label)
	}

	if c.MinLen >= c.MaxLen {
		return fmt.Errorf("field %s minLen shoud smaller than maxLen", c.Label)
	}

	return nil
}

func (c *ChartConfig) validateEnum() error {
	if c.Min != 0 || c.Max != 0 || c.MinLen != 0 || c.MaxLen != 0 {
		return fmt.Errorf("enum field %s doesn't support min, max, minLen and maxLen", c.Label)
	}

	if len(c.ValidValues) == 0 {
		return fmt.Errorf("enum field %s must set validValues", c.Label)
	}

	return nil
}

func (cs ChartConfigs) checkCriteria(configs map[string]interface{}) error {
	for _, c := range cs {
		if err := c.checkCriteria(configs); err != nil {
			return err
		}
	}

	return nil
}

func (c *ChartConfig) checkCriteria(configs map[string]interface{}) error {
	jsonKeys := strings.Split(c.JsonKey, ".")
	keyValues := configs
	var value interface{}
	for _, jsonKey := range jsonKeys {
		values, ok := keyValues[jsonKey]
		if ok == false {
			break
		}

		keyValues, ok = values.(map[string]interface{})
		if ok == false {
			value = values
			break
		}
	}

	if value == nil {
		if c.Required {
			return fmt.Errorf("field %s is required", c.Label)
		} else {
			return nil
		}
	}

	var err error
	switch c.Type {
	case TypeInt:
		err = c.checkInt(value)
	case TypeString:
		err = c.checkStr(value)
	case TypeEnum:
		err = c.checkEnum(value)
	}

	return err
}

func (c *ChartConfig) checkInt(value interface{}) error {
	val, ok := value.(float64)
	if ok == false {
		return fmt.Errorf("field %s is integer but get %s", c.Label, reflect.TypeOf(value).Kind())
	}

	if c.Min == 0 && c.Max == 0 {
		return nil
	}

	intVal := int(val)
	if intVal < c.Min || intVal >= c.Max {
		return fmt.Errorf("field %s value %d exceed the range limit [%d:%d)", c.Label, intVal, c.Min, c.Max)
	}

	return nil
}

func (c *ChartConfig) checkStr(value interface{}) error {
	strVal, ok := value.(string)
	if ok == false {
		return fmt.Errorf("field %s is string but get %s", c.Label, reflect.TypeOf(value).Kind())
	}

	if c.MinLen == 0 && c.MaxLen == 0 {
		return nil
	}

	if len(strVal) < c.MinLen || len(strVal) >= c.MaxLen {
		return fmt.Errorf("field %s value len %d exceed the range limit [%d:%d)", c.Label, len(strVal), c.MinLen, c.MaxLen)
	}

	return nil
}

func (c *ChartConfig) checkEnum(value interface{}) error {
	if slice, ok := value.([]interface{}); ok {
		for _, sliceValue := range slice {
			if err := c.checkEnumValue(sliceValue); err != nil {
				return err
			}
		}

		return nil
	}

	return c.checkEnumValue(value)
}

func (c *ChartConfig) checkEnumValue(value interface{}) error {
	strVal, ok := value.(string)
	if ok == false {
		return fmt.Errorf("field %s is enum and only support string value but get %s", c.Label, reflect.TypeOf(value).Kind())
	}

	if slice.SliceIndex(c.ValidValues, strVal) == -1 {
		return fmt.Errorf("field %s %s isn't included in validValues %v", c.Label, strVal, c.ValidValues)
	}

	return nil
}
