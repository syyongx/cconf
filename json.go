package cconf

import (
	"io/ioutil"
	"encoding/json"
)

// load reads and parses a special format file.
func loadJSON(file string, data interface{}) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(bytes, data); err != nil {
		return err
	}
	return nil
}
