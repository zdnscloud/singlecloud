package resourcedoc

import (
	"reflect"

	"github.com/zdnscloud/gorest/resource"
)

type ResourceAction struct {
	Name         string                    `json:"name"`
	Input        ResourceField             `json:"input,omitempty"`
	Output       ResourceField             `json:"output,omitempty"`
	SubResources map[string]ResourceFields `json:"subResources,omitempty"`
}

func genActions(kind resource.ResourceKind) ([]ResourceAction, error) {
	resourceActions := make([]ResourceAction, 0)
	for _, action := range kind.GetActions() {
		resourceAction := ResourceAction{
			Name:         action.Name,
			SubResources: make(map[string]ResourceFields),
		}
		if action.Input != nil {
			resourceField, err := genResourceFieldAndSubResources(resourceAction.SubResources, action.Input)
			if err != nil {
				return nil, err
			}
			resourceAction.Input = resourceField
		}
		if action.Output != nil {
			resourceField, err := genResourceFieldAndSubResources(resourceAction.SubResources, action.Output)
			if err != nil {
				return nil, err
			}
			resourceAction.Output = resourceField
		}
		resourceActions = append(resourceActions, resourceAction)
	}
	return resourceActions, nil
}

func genResourceFieldAndSubResources(subResources map[string]ResourceFields, data interface{}) (ResourceField, error) {
	var tag reflect.StructTag
	typ := reflect.TypeOf(data)
	resourceField, err := buildResourceField(typ, tag)
	if err != nil {
		return ResourceField{}, err
	}
	if t := getStructType(typ); t != nil {
		resourceFields, err := buildResourceFields(subResources, t)
		if err != nil {
			return ResourceField{}, err
		}
		subResources[LowerFirstCharacter(t.Name())] = resourceFields
	}
	return resourceField, nil
}
