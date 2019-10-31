package kvzoo

import (
	"fmt"
	"path"
	"strings"
)

const (
	MaxNestTableCount = 10
	Root              = "/"
)

type TableName string

func NewTableName(name string) (TableName, error) {
	if err := checkTableNameValid(name); err != nil {
		return "", err
	}
	return TableName(name), nil
}

func TableNameFromSegments(segments ...string) (TableName, error) {
	if len(segments) > MaxNestTableCount {
		return "", fmt.Errorf("too many sub tables")
	}
	for _, seg := range segments {
		if len(seg) == 0 {
			return "", fmt.Errorf("name of sub table should not be empty")
		}
		if strings.IndexByte(seg, '/') != -1 {
			return "", fmt.Errorf("name of sub table should not include slash")
		}
	}
	return TableName(path.Join(append([]string{Root}, segments...)...)), nil
}

func checkTableNameValid(tableName string) error {
	if strings.HasPrefix(tableName, "/") == false {
		return fmt.Errorf("table name should begin with /")
	}

	segs := segments(tableName)
	segCount := len(segs)
	if segCount == 0 {
		return fmt.Errorf("table name should not be empty")
	} else if segCount > MaxNestTableCount {
		return fmt.Errorf("too many sub tables")
	}

	for _, seg := range segs {
		if len(seg) == 0 {
			return fmt.Errorf("name of sub table should not be empty")
		}
	}

	return nil
}

func (tn TableName) Segments() []string {
	return segments(string(tn))
}

func segments(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func (tn TableName) Parent() (TableName, error) {
	parent := path.Dir(string(tn))
	if parent == Root {
		return "", fmt.Errorf("first level table has no parent")
	}
	return TableName(parent), nil
}

func (tn TableName) IsParent(child TableName) bool {
	return strings.HasPrefix(string(child), string(tn))
}
