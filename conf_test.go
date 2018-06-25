package cconf

import (
	"testing"
	"reflect"
)

func TestConf(t *testing.T) {
	c := New()
	c.Load("./testdata/app.json")
	name := c.GetString("name")
	equal(t, "cconf", name)
	email := c.GetString("ext.email")
	equal(t, "syyong.x@gmail.com", email)
	version := c.GetFloat("version", 2.0)
	equal(t, 0.1, version)
}

// Expected to be equal.
func equal(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", expected, reflect.TypeOf(expected), actual, reflect.TypeOf(actual))
	}
}
