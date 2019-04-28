package serializer

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/zdnscloud/cement/reflector"
	"github.com/zdnscloud/cement/stringtool"
)

type Serializer struct {
	nameAndObjs map[string]reflect.Value
}

func NewSerializer() *Serializer {
	return &Serializer{
		nameAndObjs: make(map[string]reflect.Value),
	}
}

func (s *Serializer) Register(o interface{}) error {
	if isStructPointer(o) == false {
		return fmt.Errorf("register non struct pointer")
	}

	s.nameAndObjs[ObjName(o)] = reflect.ValueOf(o).Elem()
	return nil
}

func isStructPointer(o interface{}) bool {
	val := reflect.ValueOf(o)
	return val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct
}

func ObjName(o interface{}) string {
	sn, _ := reflector.StructName(o)
	return stringtool.ToSnake(sn)
}

type objInJson struct {
	Name   string          `json:"name"`
	Object json.RawMessage `json:"object"`
}

func (s *Serializer) Encode(o interface{}) ([]byte, error) {
	if isStructPointer(o) == false {
		return nil, fmt.Errorf("encode non struct pointer")
	}

	name := ObjName(o)
	if _, ok := s.nameAndObjs[name]; ok == false {
		return nil, fmt.Errorf("unknown obj %v", name)
	}

	type objWrapper struct {
		Name   string      `json:"name"`
		Object interface{} `json:"object"`
	}

	j, _ := json.Marshal(objWrapper{
		Name:   name,
		Object: o,
	})
	return j, nil
}

func (s *Serializer) Decode(raw []byte) (interface{}, error) {
	var oj objInJson
	err := json.Unmarshal(raw, &oj)
	if err != nil {
		return nil, err
	}
	return s.DecodeType(oj.Name, []byte(oj.Object))
}

func (s *Serializer) DecodeType(typ string, raw []byte) (interface{}, error) {
	val, ok := s.nameAndObjs[typ]
	if ok == false {
		return nil, fmt.Errorf("unknown object %v\n", typ)
	}

	p := reflect.New(val.Type())
	p.Elem().Set(val)
	if raw != nil {
		err := json.Unmarshal(raw, p.Interface())
		if err != nil {
			return nil, fmt.Errorf("%s unmarshal to %v failed [%v]", string(raw), p.Interface(), err)
		}
	}
	return p.Interface(), nil
}

func (s *Serializer) Fill(raw []byte, out interface{}) error {
	if isStructPointer(out) == false {
		return fmt.Errorf("fill to non struct pointer")
	}

	var oj objInJson
	err := json.Unmarshal(raw, &oj)
	if err != nil {
		return err
	}

	if oj.Name != ObjName(out) {
		return fmt.Errorf("object type %v isn't match with target %v\n", oj.Name, ObjName(out))
	}

	err = json.Unmarshal([]byte(oj.Object), out)
	if err != nil {
		return err
	}
	return nil
}
