# cconf
[![MIT licensed][3]][4]

[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE

## Introduction
CConf is a Go package for handling configurations in Go applications. cconf references ozzo-config, but higher performance.

## Download & Install
```shell
go get github.com/syyongx/cconf
```

## Features
 1. loading configuration file, default JSON.
 
## Requirements
Go 1.2 or above. 
 
## Apis
```
New() *Conf
RegisterLoadFunc(typ string, fn loadFunc)
Load(files ...string) error
LoadWithPattern(pattern string) error

Set(key string, val interface{}) error
Get(key string, def ...interface{}) interface{}
GetString(key string, def ...string) string
GetInt(key string, def ...int) int
GetInt64(key string, def ...int64) int64
GetFloat(key string, def ...float64) float64
GetBool(key string, def ...bool) bool

SetStore(data ...interface{})
GetStore() interface{}
```

## Usage
```
import github.com/syyongx/cconf

func main() {
    c := cconf.New()
    age := c.GetInt("age", 18)
    name := c.Get("name").(string)
    c.Set("email", "default@default.com")
    email := c.GetString("email")
}
```

## LICENSE
cconf source code is licensed under the [MIT](https://github.com/syyongx/cconf/blob/master/LICENSE) Licence.
