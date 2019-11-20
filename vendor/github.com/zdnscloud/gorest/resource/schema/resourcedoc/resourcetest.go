package resourcedoc

import (
	"encoding/json"
	"github.com/zdnscloud/gorest/resource"
)

func (s TestStruct) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{}
}

type TestStruct struct {
	resource.ResourceBase                     `json:",inline"`
	Name                                      string              `json:"name"`
	Int                                       int                 `json:"int"`
	Uint                                      uint                `json:"uint"`
	Int8                                      int8                `json:"int8"`
	Uint8                                     uint8               `json:"uint8"`
	Int32                                     int32               `json:"int32"`
	Uint32                                    uint32              `json:"uint32"`
	MapStringString                           map[string]string   `json:"mapStringString"`
	MapStringInt                              map[string]int      `json:"mapStringInt"`
	MapStringInt8                             map[string]int8     `json:"mapStringInt8"`
	MapStringStruct                           map[string]Struct   `json:"mapStringStruct"`
	MapStringStructPtr                        map[string]*Struct  `json:"mapStringStructPtr"`
	Bool                                      bool                `json:"bool"`
	SliceString                               []string            `json:"sliceString"`
	SliceInt                                  []int               `json:"sliceInt"`
	SliceInt8                                 []int8              `json:"sliceInt8"`
	SliceUint32                               []uint32            `json:"sliceUint32"`
	SliceBool                                 []bool              `json:"sliceBool"`
	SliceStruct                               []Struct            `json:"sliceStruct"`
	SliceStructPtr                            []*Struct           `json:"sliceStructPtr"`
	SliceMapStringString                      []map[string]string `json:"sliceMapStringString"`
	SliceMapStringStruct                      []map[string]Struct `json:"sliceMapStringStruct"`
	StructPtr                                 *Struct             `json:"structPtr"`
	Date                                      resource.ISOTime    `json:"date"`
	Json                                      json.RawMessage     `json:"configs"`
	Required                                  string              `json:"required" rest:"required=true"`
	Options                                   string              `json:"options" rest:"options=lvm|cephfs"`
	RequiredAndOptions                        string              `json:"requiredAndOptions" rest:"required=true,options=lvm|cephfs"`
	DescriptionOnly                           string              `json:"descriptionOnly" rest:"description=readonly"`
	DescriptionImmutable                      string              `json:"descriptionImmutable" rest:"description=immutable"`
	RequiredAndDescriptionOnly                string              `json:"requiredAndDescriptionOnly" rest:"required=true,description=readonly"`
	RequiredAndOptionsAndDescriptionImmutable string              `json:"requiredAndOptionsAndDescriptionImmutable" rest:"required=true,options=lvm|cephfs,description=immutable"`
}

type Struct struct {
	Name string
	Id   int
	Str  Struct1
}
type Struct1 struct {
	Name string
	Str  Struct2
}
type Struct2 struct {
	Id int
}
