package cconf

import (
	"reflect"
	"testing"
)

func TestConf(t *testing.T) {
	c := New()
	c.RegisterLoadFunc("json", loadJSON)
	c.Load("./testdata/app.json")
	name := c.GetString("name")
	equal(t, "cconf", name)
	ext := c.Get("ext")
	equal(t, map[string]interface{}{"email": "syyong.x@gmail.com", "author": "syyong.x"}, ext)
	email := c.GetString("ext.email")
	equal(t, "syyong.x@gmail.com", email)
	version := c.GetFloat("version", 2.0)
	equal(t, 0.1, version)
}

func TestPopulate(t *testing.T) {

}

// Expected to be equal.
func equal(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", expected, reflect.TypeOf(expected), actual, reflect.TypeOf(actual))
	}
}
