package cconf

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// load file function
type loadFunc func(string, interface{}) error

// DefaultSeparator default separator.
var DefaultSeparator = "."

// DefaultLoadFuncs default load functions.
var DefaultLoadFuncs = map[string]loadFunc{"json": loadJSON}

// Conf conf
type Conf struct {
	Separator string
	LoadFuncs map[string]loadFunc
	types     map[string]reflect.Value
	store     reflect.Value
	cache     map[string]interface{}
}

// New returns an instance of the Conf.
func New() *Conf {
	return &Conf{
		Separator: DefaultSeparator,
		LoadFuncs: DefaultLoadFuncs,
		types:     make(map[string]reflect.Value),
		cache:     make(map[string]interface{}),
	}
}

// RegisterLoadFunc register load function.
// like:
// RegisterLoadFunc("toml", loadTOML)
// RegisterLoadFunc("yaml", loadYAML)
func (c *Conf) RegisterLoadFunc(typ string, fn loadFunc) {
	c.LoadFuncs[typ] = fn
}

// Load loads configuration data from one or multiple files.
func (c *Conf) Load(files ...string) error {
	defer func() {
		// Reset cache.
		c.cache = make(map[string]interface{})
	}()
	for _, file := range files {
		typ := strings.TrimLeft(filepath.Ext(file), ".")
		if fn, ok := c.LoadFuncs[typ]; ok {
			var data interface{}
			if err := fn(file, &data); err != nil {
				return err
			}
			c.store = merge(c.store, reflect.ValueOf(data))
		} else {
			return errors.New("please register " + typ + " type loading function")
		}
	}
	return nil
}

// LoadWithPattern loads configuration data from the names of all files matching pattern or nil.
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// testdata/*.json (assuming the Separator is '/').
func (c *Conf) LoadWithPattern(pattern string) error {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	return c.Load(files...)
}

// Set sets the configuration value at the specified path.
func (c *Conf) Set(key string, val interface{}) error {
	if !c.store.IsValid() {
		c.store = reflect.ValueOf(make(map[string]interface{}))
		c.cache = make(map[string]interface{})
	}
	defer delete(c.cache, key)

	store := c.store
	segs := strings.Split(key, c.Separator)
	length := len(segs)
	for i := 0; i < length; i++ {
		switch store.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
		default:
			return &ConfigKeyError{strings.Join(segs[:i+1], c.Separator), fmt.Sprintf("got %v instead of a map, array, or slice", store.Kind())}
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
			return &ConfigKeyError{strings.Join(segs[:i+1], c.Separator), err.Error()}
		}

		store = reflect.ValueOf(newMap)
	}

	return nil
}

// Get config
func (c *Conf) Get(key string, def ...interface{}) interface{} {
	var v interface{}
	if len(def) > 0 {
		v = def[0]
	}
	// take priority from the cache.
	if cv, ok := c.cache[key]; ok {
		if cv == nil {
			return v
		}
		return cv
	}
	store := c.store
	segs := strings.Split(key, c.Separator)
	length := len(segs)
	for i := 0; i < length-1; i++ {
		if store = getElement(store, segs[i]); !store.IsValid() {
			c.cache[key] = nil
			return v
		}
	}
	val := getElement(store, segs[length-1])
	if !val.IsValid() {
		c.cache[key] = nil
		return v
	}

	// convert the value to the same type as the default value.
	if tv := reflect.ValueOf(v); tv.IsValid() {
		if val.Type().ConvertibleTo(tv.Type()) {
			c.cache[key] = val.Convert(tv.Type()).Interface()
			return c.cache[key]
		}
		// unable to convert: return the default value.
		c.cache[key] = nil
		return v
	}

	c.cache[key] = val.Interface()
	return c.cache[key]
}

// GetString returns a string.
func (c *Conf) GetString(key string, def ...string) string {
	var v string
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(string)
}

// GetInt returns an int
func (c *Conf) GetInt(key string, def ...int) int {
	var v int
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(int)
}

// GetInt64 returns an int64
func (c *Conf) GetInt64(key string, def ...int64) int64 {
	var v int64
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(int64)
}

// GetFloat returns an float
func (c *Conf) GetFloat(key string, def ...float64) float64 {
	var v float64
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(float64)
}

// GetBool returns a bool
func (c *Conf) GetBool(key string, def ...bool) bool {
	var v bool
	if len(def) > 0 {
		v = def[0]
	}
	return c.Get(key, v).(bool)
}

// GetStore returns the complete configuration store.
// Nil will be returned if the configuration has never been loaded before.
func (c *Conf) GetStore() interface{} {
	if c.store.IsValid() {
		return c.store.Interface()
	}
	return nil
}

// SetStore sets the configuration data.
//
// If multiple configurations are given, they will be merged sequentially. The following rules are taken
// when merging two configurations C1 and C2:
// A). If either C1 or C2 is not a map, replace C1 with C2;
// B). Otherwise, add all key-value pairs of C2 to C1; If a key of C2 is also found in C1,
// merge the corresponding values in C1 and C2 recursively.
//
// Note that this method will clear any existing configuration data.
func (c *Conf) SetStore(data ...interface{}) {
	c.store = reflect.Value{}
	for _, d := range data {
		c.store = merge(c.store, reflect.ValueOf(d))
	}
	c.cache = make(map[string]interface{})
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
