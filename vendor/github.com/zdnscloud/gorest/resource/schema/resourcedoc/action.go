package resourcedoc

import (
	"fmt"
	"reflect"

	"github.com/zdnscloud/gorest/resource"
)

type ResourceAction struct {
	Name         string                    `json:"name"`
	Input        ResourceFields            `json:"input,omitempty"`
	Output       ResourceFields            `json:"output,omitempty"`
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
			if t := getStructType(reflect.TypeOf(action.Input)); t != nil {
				resourceFields, err := buildResourceFields(resourceAction.SubResources, t)
				if err != nil {
					return nil, err
				}
				resourceAction.Input = resourceFields
			} else {
				return nil, fmt.Errorf("kind %s action %s input must be struct", reflect.TypeOf(kind).Name(), action.Name)
			}
		}
		if action.Output != nil {
			if t := getStructType(reflect.TypeOf(action.Output)); t != nil {
				resourceFields, err := buildResourceFields(resourceAction.SubResources, t)
				if err != nil {
					return nil, err
				}
				resourceAction.Output = resourceFields
			} else {
				return nil, fmt.Errorf("kind %s action %s output must be struct", reflect.TypeOf(kind).Name(), action.Name)
			}
		}
		resourceActions = append(resourceActions, resourceAction)
	}
	return resourceActions, nil
}
