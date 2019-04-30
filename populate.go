package cconf

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// ConfigKeyError describes a key which cannot be used to set a configuration value.
type ConfigKeyError struct {
	Key     string
	Message string
}

// Error
func (ck *ConfigKeyError) Error() string {
	return fmt.Sprintf("%q is not a valid key: %v", ck.Key, ck.Message)
}

// ConfigValueError describes a configuration that cannot be used to configure a target value
type ConfigValueError struct {
	Key     string // path to the configuration value
	Message string // the detailed error message
}

// Error returns the error message represented by ConfigValueError
func (cv *ConfigValueError) Error() string {
	key := strings.Trim(cv.Key, ".")
	return fmt.Sprintf("%q points to an inappropriate configuration value: %v", key, cv.Message)
}

// ConfigTargetError describes a target value that cannot be configured
type ConfigTargetError struct {
	Value reflect.Value
}

// Error returns the error message represented by ConfigTargetError
func (ct *ConfigTargetError) Error() string {
	if ct.Value.Kind() != reflect.Ptr {
		return "Unable to configure a non-pointer"
	}
	if ct.Value.IsNil() {
		return "Unable to configure a nil pointer"
	}
	return ""
}

// ProviderError describes a provider that was not appropriate for a type
type ProviderError struct {
	Value reflect.Value
}

// Error returns the error message represented by ProviderError
func (pe *ProviderError) Error() string {
	if pe.Value.Kind() != reflect.Func {
		return fmt.Sprintf("The provider should be a function, got %v", pe.Value.Kind())
	}
	if pe.Value.Type().NumOut() != 1 {
		return fmt.Sprintf("The provider should have a single output, got %v", pe.Value.Type().NumOut())
	}
	return ""
}

// Register associates a type name with a provider that creates an instance of the type.
// The provider must be a function with a single output.
// Register is mainly needed when calling Configure() to configure an object and create
// new instances of the specified types.
func (c *Conf) Register(name string, provider interface{}) error {
	v := reflect.ValueOf(provider)
	if v.Kind() != reflect.Func || v.Type().NumOut() != 1 {
		return &ProviderError{v}
	}
	c.types[name] = v
	return nil
}

// Populate populate.
func (c *Conf) Populate(v interface{}, key ...string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return &ConfigTargetError{val}
	}
	f := ""
	config := c.store
	if len(key) > 0 {
		d := c.Get(key[0])
		if d == nil {
			return &ConfigKeyError{key[0], "no configuration value was found"}
		}
		f = key[0]
		config = reflect.ValueOf(d)
	}
	return c.populate(val, config, f)
}

// indirect
func indirect(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

// populate populate the value with the configuration.
func (c *Conf) populate(v, config reflect.Value, key string) error {
	// get the concrete value, may allocate space if needed.
	v = indirect(v)

	if !v.IsValid() {
		return nil
	}
	for config.Kind() == reflect.Interface || config.Kind() == reflect.Ptr {
		config = config.Elem()
	}

	switch config.Kind() {
	case reflect.Array, reflect.Slice:
		return c.populateArray(v, config, key)
	case reflect.Map:
		switch v.Kind() {
		case reflect.Interface:
			return c.populateInterface(v, config, key)
		case reflect.Struct:
			return c.populateStruct(v, config, key)
		case reflect.Map:
			return c.populateMap(v, config, key)
		default:
			return &ConfigValueError{key, "a map cannot be used to configure " + v.Type().String()}
		}
	default:
		return c.populateScalar(v, config, key)
	}
}

// populateArray
func (c *Conf) populateArray(v, config reflect.Value, key string) error {
	vkind := v.Kind()

	// nil interface
	if vkind == reflect.Interface && v.NumMethod() == 0 {
		v.Set(config)
		return nil
	}

	if vkind != reflect.Array && vkind != reflect.Slice {
		return &ConfigValueError{key, fmt.Sprintf("%v cannot be used to configure %v", config.Type(), v.Type())}
	}

	n := config.Len()

	// grow slice if it's smaller than the config array
	if vkind == reflect.Slice && v.Cap() < n {
		t := reflect.MakeSlice(v.Type(), n, n)
		reflect.Copy(t, v)
		v.Set(t)
	}

	if n > v.Cap() {
		n = v.Cap()
	}
	for i := 0; i < n; i++ {
		if err := c.populate(v.Index(i), config.Index(i), key+"."+strconv.Itoa(i)); err != nil {
			return err
		}
	}

	if n < v.Len() {
		if vkind == reflect.Array {
			// Array.  Zero the rest.
			z := reflect.Zero(v.Type().Elem())
			for i := n; i < v.Len(); i++ {
				v.Index(i).Set(z)
			}
		} else {
			v.SetLen(n)
		}
	}

	return nil
}

// populateMap
func (c *Conf) populateMap(v, config reflect.Value, key string) error {
	// map must have string kind
	t := v.Type()
	if v.IsNil() {
		v.Set(reflect.MakeMap(t))
	}

	for _, k := range config.MapKeys() {
		elemType := v.Type().Elem()
		mapElem := reflect.New(elemType).Elem()
		if err := c.populate(mapElem, mapIndex(config, k), key+"."+k.String()); err != nil {
			return err
		}
		v.SetMapIndex(k.Convert(v.Type().Key()), mapElem)
	}

	return nil
}

// the "type" field name
var typeKey = reflect.ValueOf("type")

// populateStruct
func (c *Conf) populateStruct(v, config reflect.Value, key string) error {
	for _, k := range config.MapKeys() {
		if k.String() == typeKey.String() {
			continue
		}
		key = key + "." + k.String()
		field := v.FieldByName(k.Interface().(string))
		if !field.IsValid() {
			return &ConfigValueError{key, fmt.Sprintf("field %v not found in struct %v", k.String(), v.Type())}
		}
		if !field.CanSet() {
			return &ConfigValueError{key, fmt.Sprintf("field %v cannot be set", k.String())}
		}
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}
		if err := c.populate(field, mapIndex(config, k), key); err != nil {
			return err
		}
	}

	return nil
}

// populateInterface
func (c *Conf) populateInterface(v, config reflect.Value, key string) error {
	// nil interface
	if v.NumMethod() == 0 {
		v.Set(config)
		return nil
	}

	tk := mapIndex(config, typeKey)
	if !tk.IsValid() {
		return &ConfigValueError{key, "missing the type element"}
	}
	if tk.Kind() != reflect.String {
		return &ConfigValueError{key, "type must be a string"}
	}

	builder, ok := c.types[tk.String()]
	if !ok {
		return &ConfigValueError{key, fmt.Sprintf("type %q is unknown", tk.String())}
	}

	object := builder.Call([]reflect.Value{})[0]

	s := indirect(object)
	if !s.Addr().Type().Implements(v.Type()) {
		return &ConfigValueError{key, fmt.Sprintf("%v does not implement %v", s.Type(), v.Type())}
	}
	v.Set(object)

	return c.populateStruct(s, config, key)
}

// populateScalar
func (c *Conf) populateScalar(v, config reflect.Value, key string) error {
	if !config.IsValid() {
		switch v.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			v.Set(reflect.Zero(v.Type()))
			// otherwise, ignore null for primitives/string
		}
		return nil
	}

	// nil interface
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
		v.Set(config)
		return nil
	}

	if config.Type().ConvertibleTo(v.Type()) {
		v.Set(config.Convert(v.Type()))
		return nil
	}

	return &ConfigValueError{key, fmt.Sprintf("%v cannot be used to configure %v", config.Type(), v.Type())}
}
