package resource

import (
	"reflect"
	"strings"
	"time"

	"github.com/zdnscloud/gorest/util"
)

type ResourceLinkType string
type ResourceLink string //url schema + host + path

const (
	SelfLink       ResourceLinkType = "self"
	UpdateLink     ResourceLinkType = "update"
	RemoveLink     ResourceLinkType = "remove"
	CollectionLink ResourceLinkType = "collection"
)

type Resource interface {
	GetParent() Resource
	SetParent(Resource)

	GetID() string
	SetID(string)

	GetLinks() map[ResourceLinkType]ResourceLink
	SetLinks(map[ResourceLinkType]ResourceLink)

	GetCreationTimestamp() time.Time
	SetCreationTimestamp(time.Time)

	GetSchema() Schema
	SetSchema(Schema)

	SetType(string)
	//return resource kind name
	GetType() string

	GetAction() *Action
	SetAction(*Action)
}

//struct implement ResourceKind
//struct pointer should implement Resource
type ResourceKind interface {
	GetParents() []ResourceKind
	//return the default resource if the related field
	//isn't speicified in json data
	//NOTE: default field shouldn't include map
	//json unmarshal will merge map, in this case
	//when real data is provided, it will merge with
	//default value
	CreateDefaultResource() Resource
	CreateAction(name string) *Action
}

//lowercase singluar
//eg: type Node struct -> node
func DefaultKindName(kind ResourceKind) string {
	return strings.ToLower(reflect.TypeOf(kind).Name())
}

//resource name is lowercase, plural word
//eg: type Node struct -> nodes
func DefaultResourceName(kind ResourceKind) string {
	gt := reflect.TypeOf(kind)
	return util.GuessPluralName(strings.ToLower(gt.Name()))
}
