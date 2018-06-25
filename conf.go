package cconf

import (
	"strings"
	"reflect"
	"fmt"
	"strconv"
)

var DefaultSeparator = "."

// ConfigKeyError describes a key which cannot be used to set a configuration value.
type ConfigKeyError struct {
	Key     string
	Message string
}

// Error
func (ce *ConfigKeyError) Error() string {
	return fmt.Sprintf("%q is not a valid path: %v", ce.Key, ce.Message)
}

// load file function
type loadFunc func(string, interface{}) error

// Conf
type Conf struct {
	Separator string
	store     reflect.Value
	types     map[string]reflect.Value
	loadFunc  loadFunc
}

// New
func New(lf loadFunc) *Conf {
	return &Conf{
		Separator: DefaultSeparator,
		types:     make(map[string]reflect.Value),
		loadFunc:  lf,
	}
}

// Get config
func (c *Conf) Get(key string, def ...interface{}) interface{} {
	var v interface{}
	if len(def) > 0 {
		v = def[0]
	}
	store := c.store
	segs := strings.Split(key, c.Separator)
	length := len(segs)
	for i := 0; i < length-1; i++ {
		if store = getElement(store, segs[i]); !store.IsValid() {
			return v
		}
	}
	val := getElement(store, segs[length-1])
	if !val.IsValid() {
		return v
	}

	// convert the value to the same type as the default value
	if tv := reflect.ValueOf(v); tv.IsValid() {
		if val.Type().ConvertibleTo(tv.Type()) {
			return val.Convert(tv.Type()).Interface()
		}
		// unable to convert: return the default value
		return v
	}

	return val.Interface()
}

// Set sets the configuration value at the specified path.
func (c *Conf) Set(key string, val interface{}) error {
	if !c.store.IsValid() {
		c.store = reflect.ValueOf(make(map[string]interface{}))
	}

	store := c.store
	segs := strings.Split(key, ".")
	length := len(segs)
	for i := 0; i < length; i++ {
		switch store.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
		default:
			return &ConfigKeyError{strings.Join(segs[:i+1], "."), fmt.Sprintf("got %v instead of a map, array, or slice", store.Kind())}
		}

		if i == length-1 {
			if err := setElement(store, segs[i], val); err != nil {
				return &ConfigKeyError{key, err.Error()}
			}
			return nil
		}

		e := getElement(store, segs[i])
		if e.IsValid() {
			store = e
			continue
		}

		newMap := make(map[string]interface{})
		if err := setElement(store, segs[i], newMap); err != nil {
			return &ConfigKeyError{strings.Join(segs[:i+1], "."), err.Error()}
		}

		store = reflect.ValueOf(newMap)
	}

	return nil
}

// GetString
func (c *Conf) GetString(key string, def ...string) string {
	var v string
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(string)
}

// GetInt
func (c *Conf) GetInt(key string, def ...int) int {
	var v int
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(int)
}

// GetInt64
func (c *Conf) GetInt64(key string, def ...int64) int64 {
	var v int64
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(int64)
}

// GetFloat
func (c *Conf) GetFloat(key string, def ...float64) float64 {
	var v float64
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(float64)
}

// GetBool
func (c *Conf) GetBool(key string, def ...bool) bool {
	var v bool
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(bool)
}

// Store returns the complete configuration store.
// Nil will be returned if the configuration has never been loaded before.
func (c *Conf) Store() interface{} {
	if c.store.IsValid() {
		return c.store.Interface()
	}
	return nil
}

// SetStore sets the configuration store.
func (c *Conf) SetStore(data ...interface{}) {
	c.store = reflect.Value{}
	for _, d := range data {
		c.store = merge(c.store, reflect.ValueOf(d))
	}
}

// Load loads configuration data from one or multiple files.
func (c *Conf) Load(files ...string) error {
	for _, file := range files {
		var data interface{}
		if err := c.loadFunc(file, &data); err != nil {
			return err
		}
		c.store = merge(c.store, reflect.ValueOf(data))
	}
	return nil
}

// mapIndex
func mapIndex(data reflect.Value, index reflect.Value) reflect.Value {
	v := data.MapIndex(index)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

// merge
func merge(v1, v2 reflect.Value) reflect.Value {
	if v1.Kind() != reflect.Map || v2.Kind() != reflect.Map || !v1.IsValid() {
		return v2
	}

	for _, key := range v2.MapKeys() {
		e1 := mapIndex(v1, key)
		e2 := mapIndex(v2, key)
		if e1.Kind() == reflect.Map && e2.Kind() == reflect.Map {
			e2 = merge(e1, e2)
		}
		v1.SetMapIndex(key, e2)
	}

	return v1
}

// getElement returns the element value of a map, array, or slice at the specified index.
func getElement(v reflect.Value, seg string) reflect.Value {
	switch v.Kind() {
	case reflect.Map:
		return mapIndex(v, reflect.ValueOf(seg))
	case reflect.Slice, reflect.Array:
		if i, err := strconv.Atoi(seg); err == nil {
			if i >= 0 && i < v.Len() {
				v = v.Index(i)
				for v.Kind() == reflect.Interface {
					v = v.Elem()
				}
				return v
			}
		}
	}

	return reflect.Value{}
}

// setElement ses the element value of a map, array, or slice at the specified index.
func setElement(data reflect.Value, seg string, v interface{}) error {
	val := reflect.ValueOf(v)

	switch data.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(seg)
		data.SetMapIndex(key, val)
	case reflect.Slice, reflect.Array:
		i, err := strconv.Atoi(seg)
		if err != nil || i < 0 {
			return fmt.Errorf("%v is not a valid array or slice index", seg)
		}
		if data.Kind() == reflect.Slice {
			if i >= data.Cap() {
				return fmt.Errorf("%v is out of the slice index bound", seg)
			}
			data.SetLen(i + 1)
		} else if i >= data.Cap() {
			return fmt.Errorf("%v is out of the array index bound", seg)
		}
		data.Index(i).Set(val)
	}

	return nil
}
