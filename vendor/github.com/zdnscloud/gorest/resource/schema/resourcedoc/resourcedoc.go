package resourcedoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	slice "github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/gorest/util"
)

const (
	requiredTag    = "required"
	optionsTag     = "options="
	descriptionTag = "description="
	docFileSuffix  = ".json"
	ignoreField    = "ResourceBase"
	ignoreJsonFlag = "inline"
	ignoreJsonName = "-"
	supportKeyType = "string"
)

type ResourceDocument struct {
	ResourceType      string                    `json:"resourceType,omitempty"`
	CollectionName    string                    `json:"collectionName,omitempty"`
	ParentResources   []string                  `json:"parentResources,omitempty"`
	GoStructName      string                    `json:"goStructName,omitempty"`
	ResourceFields    ResourceFields            `json:"resourceFields,omitempty"`
	SubResources      map[string]ResourceFields `json:"subResources,omitempty"`
	ResourceMethods   []resource.HttpMethod     `json:"resourceMethods,omitempty"`
	CollectionMethods []resource.HttpMethod     `json:"collectionMethods,omitempty"`
}

type ResourceFields map[string]ResourceField

type ResourceField struct {
	Type        string   `json:"type,omitempty"`
	ValidValues []string `json:"validValues,omitempty"`
	ElemType    string   `json:"elemType,omitempty"`
	KeyType     string   `json:"keyType,omitempty"`
	ValueType   string   `json:"valueType,omitempty"`
	Description []string `json:"description,omitempty"`
}

func NewResourceDocument(name string, kind resource.ResourceKind, handler resource.Handler, parents []string) (*ResourceDocument, error) {
	resource := &ResourceDocument{
		ResourceType:      name,
		CollectionName:    util.GuessPluralName(name),
		ParentResources:   parents,
		GoStructName:      reflect.TypeOf(kind).Name(),
		SubResources:      make(map[string]ResourceFields),
		ResourceMethods:   resource.GetResourceMethods(handler),
		CollectionMethods: resource.GetCollectionMethods(handler),
	}
	if resourceFields, err := buildResourceFields(resource, reflect.TypeOf(kind)); err != nil {
		return resource, fmt.Errorf("build resource %s failed, %s", name, err.Error())
	} else {
		resource.ResourceFields = resourceFields
	}
	return resource, nil
}

func (r *ResourceDocument) WriteJsonFile(targetPath string) error {
	if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
		return err
	}
	filePtr, err := os.Create(path.Join(targetPath, r.ResourceType+docFileSuffix))
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	_, err = filePtr.Write(data)
	return err
}

func buildResourceFields(resource *ResourceDocument, t reflect.Type) (ResourceFields, error) {
	resourceFields := make(map[string]ResourceField)
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		typ := t.Field(i).Type
		tag := t.Field(i).Tag
		jsonName := fieldJsonName(name, tag)
		if (strings.HasSuffix(name, ignoreField) && slice.SliceIndex(strings.Split(tag.Get("json"), ","), ignoreJsonFlag) >= 0) || jsonName == ignoreJsonName {
			continue
		}
		if resourceField, err := buildResourceField(typ, tag); err != nil {
			return resourceFields, fmt.Errorf("field %s has %s", name, err.Error())
		} else {
			resourceFields[jsonName] = resourceField
		}

		if _, ignore := getIgnoreType(typ); !ignore {
			if t := getStructType(typ); t != nil {
				if resourceFields, err := buildResourceFields(resource, t); err != nil {
					return resourceFields, err
				} else {
					resource.SubResources[LowerFirstCharacter(t.Name())] = resourceFields
				}
			}
		}
	}
	return resourceFields, nil
}

func buildResourceField(t reflect.Type, tag reflect.StructTag) (ResourceField, error) {
	typ, ignore := getIgnoreType(t)
	resourceField := ResourceField{
		Type:        typ,
		Description: parseTag(tag, false),
	}
	if !ignore {
		if valueRange := parseTag(tag, true); len(valueRange) > 0 {
			resourceField.Type = Enum
			resourceField.ValidValues = valueRange
		} else {
			resourceField.Type = getType(t)
			switch resourceField.Type {
			case Array:
				if elemType := getElemType(t); elemType == Unknow {
					return resourceField, errors.New("unsupport array elem type")
				} else {
					resourceField.ElemType = elemType
				}
			case Map:
				if valueType := getElemType(t); valueType == Unknow {
					return resourceField, errors.New("unsupport map value type")
				} else {
					resourceField.KeyType = supportKeyType
					resourceField.ValueType = valueType
				}
			case Unknow:
				return resourceField, errors.New("unsupport type")
			}
		}
	}
	return resourceField, nil
}

func parseTag(tag reflect.StructTag, isOptions bool) []string {
	var tags []string
	restTags := strings.Split(tag.Get("rest"), ",")
	for _, t := range restTags {
		if isOptions {
			if strings.HasPrefix(t, optionsTag) {
				tags = append(tags, strings.Split(strings.TrimPrefix(t, optionsTag), "|")...)
				break
			}
		} else {
			if strings.HasPrefix(t, requiredTag) {
				tags = append(tags, requiredTag)
			}
			if strings.HasPrefix(t, descriptionTag) {
				tags = append(tags, strings.TrimPrefix(t, descriptionTag))
			}
		}
	}
	return tags
}
