package resourcedoc

import (
	"encoding/json"
	"os"
	"path"
	"reflect"

	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/gorest/util"
)

type Document struct {
	ResourceType      string                             `json:"resourceType,omitempty"`
	CollectionName    string                             `json:"collectionName,omitempty"`
	ParentResources   []string                           `json:"parentResources,omitempty"`
	ResourceFields    []map[string]DocField              `json:"resourceFields,omitempty"`
	SubResources      []map[string][]map[string]DocField `json:"subResources,omitempty"`
	ResourceMethods   []resource.HttpMethod              `json:"resourceMethods,omitempty"`
	CollectionMethods []resource.HttpMethod              `json:"collectionMethods,omitempty"`
}

type DocField struct {
	Type        string   `json:"type,omitempty"`
	ValidValues []string `json:"validValues,omitempty"`
	ElemType    string   `json:"elemType,omitempty"`
	KeyType     string   `json:"keyType,omitempty"`
	ValueType   string   `json:"valueType,omitempty"`
	Description []string `json:"description,omitempty"`
}

type DocumentManager struct {
	resourceName string
	resourceKind resource.ResourceKind
	document     Document
}

func NewDocumentManager(name string, kind resource.ResourceKind, handler resource.Handler, parents []string) *DocumentManager {
	builder := NewBuilder()
	builder.BuildResource(name, reflect.TypeOf(kind))
	var resourceFields []map[string]DocField
	for _, v := range builder.GetTop() {
		resourceFields = genDocField(v)
	}
	subresources := make([]map[string][]map[string]DocField, 0)
	for _, resource := range builder.GetSub() {
		for k, v := range resource {
			subresource := make(map[string][]map[string]DocField)
			subresource[LowerFirstCharacter(k)] = genDocField(v)
			subresources = append(subresources, subresource)
		}
	}
	return &DocumentManager{
		resourceName: name,
		resourceKind: kind,
		document: Document{
			ResourceType:      LowerFirstCharacter(reflect.TypeOf(kind).Name()),
			CollectionName:    util.GuessPluralName(LowerFirstCharacter(reflect.TypeOf(kind).Name())),
			ParentResources:   parents,
			ResourceFields:    resourceFields,
			SubResources:      subresources,
			ResourceMethods:   resource.GetResourceMethods(handler),
			CollectionMethods: resource.GetCollectionMethods(handler),
		},
	}
}

func (d *DocumentManager) WriteJsonFile(targetPath string) error {
	data, err := json.MarshalIndent(d.document, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
		return err
	}
	file := path.Join(targetPath, d.document.ResourceType+".json")
	filePtr, err := os.Create(file)
	if err != nil {
		return err
	}
	filePtr.Write(data)
	return nil
}

func genDocField(fields []Field) []map[string]DocField {
	var docFields []map[string]DocField
	for _, field := range fields {
		if docField := fieldToDoc(field); len(docField) > 0 {
			docFields = append(docFields, docField)
		}
	}
	return docFields
}

func fieldToDoc(f Field) map[string]DocField {
	var typ, elemType, keyType, valueType string
	validValues := OptionsTag(f.Tag)
	if f.Special == "" {
		if len(validValues) > 0 {
			typ = Enum
		} else {
			typ = setType(f.Type)
			switch typ {
			case Array:
				elemType = setSlice(f.Type)
			case Map:
				keyType, valueType = setMap(f.Type)
			}
		}
	} else {
		typ = f.Special
	}
	field := make(map[string]DocField)
	newname := LowerFirstCharacter(fieldJsonName(f.Name, f.Tag.Get("json")))
	field[newname] = DocField{
		Type:        LowerFirstCharacter(typ),
		ElemType:    LowerFirstCharacter(elemType),
		ValidValues: validValues,
		KeyType:     LowerFirstCharacter(keyType),
		ValueType:   LowerFirstCharacter(valueType),
		Description: DescriptionTag(f.Tag),
	}
	return field
}
