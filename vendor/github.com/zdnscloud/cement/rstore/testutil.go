package rstore

import (
	"time"
)

type child struct {
	Id       string
	Name     string `sql:"uk"`
	Age      uint32
	Hobbies  []string
	Scores   []int
	Birthday time.Time
	Talented bool
}

func (t *child) Validate() error {
	return nil
}

type tuser struct {
	Id   string
	Name string `sql:"uk"`
	Age  int
	CId  int
}

func (t *tuser) Validate() error {
	return nil
}

type tuserTview struct {
	Id    string
	Tuser string `sql:"ownby"`
	Tview string `sql:"referto"`
}

func (t *tuserTview) Validate() error {
	return nil
}

type tview struct {
	Id   string
	Name string `sql:"uk"`
}

func (t *tview) Validate() error {
	return nil
}

type trr struct {
	Id    string
	Name  string `sql:"uk"`
	Tview string `sql:"referto,uk"`
	Ttl   int
}

func (t *trr) Validate() error {
	return nil
}

type tnest struct {
	Id    string
	Name  string         `sql:"uk"`
	Inner map[string]int `sql:"-"`
}

func (t *tnest) Validate() error {
	return nil
}

type trrr struct {
	Id   string
	Name string `sql:"suk"`
	Age  int    `sql:"suk"`
}

func (t *trrr) Validate() error {
	return nil
}
