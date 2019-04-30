package cconf

import (
	"encoding/json"
	"io/ioutil"
)

// load reads and parses a special format file.
func loadJSON(file string, data interface{}) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, data)
}
