package main

import (
	"strconv"
	"fmt"
	"strings"
	"github.com/Shopify/go-lua"
)

type Parser func(s string) (map[string]interface{}, error)

func wrapper(field string) func(value interface{}, err error) (map[string]interface{}, error) {
	return func(value interface{}, err error) (map[string]interface{}, error) {
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			field: value,
		}, nil
	}
}

func MakeStringParser(opts []string) (Parser, error) {
	if len(opts) != 1 {
		return nil, fmt.Errorf("expected parser option: field name")
	}

	field := wrapper(opts[0])

	return func(s string) (map[string]interface{}, error) {
		return field(s, nil)
	}, nil
}

func MakeBoolParser(opts []string) (Parser, error) {
	if len(opts) != 1 {
		return nil, fmt.Errorf("expected parser option: field name")
	}

	field := wrapper(opts[0])

	return func(s string) (map[string]interface{}, error) {
		// TODO: Specify values in config
		return field(strconv.ParseBool(s))
	}, nil
}

func MakeIntParser(opts []string) (Parser, error) {
	if len(opts) != 1 {
		return nil, fmt.Errorf("expected parser option: field name")
	}

	field := wrapper(opts[0])

	return func(s string) (map[string]interface{}, error) {
		return field(strconv.ParseInt(s, 0, 64))
	}, nil
}

func MakeFloatParser(opts []string) (Parser, error) {
	if len(opts) != 1 {
		return nil, fmt.Errorf("expected parser option: field name")
	}

	field := wrapper(opts[0])

	return func(s string) (map[string]interface{}, error) {
		return field(strconv.ParseFloat(s, 64))
	}, nil
}

func MakeLuaParser(opts []string) (Parser, error) {
	if len(opts) != 1 {
		return nil, fmt.Errorf("expected parser option: script path")
	}

	// Create lua context and load script
	l := lua.NewState()
	lua.OpenLibraries(l)
	if err := lua.DoFile(l, opts[0]); err != nil {
		return nil, err
	}

	// Remember script return value as function to call
	f := l.Top()
	if !l.IsFunction(f) {
		return nil, fmt.Errorf("script must return function")
	}

	return func(s string) (map[string]interface{}, error) {
		// Push function and parameter to call
		l.PushValue(f)
		l.PushString(s)
		err := l.ProtectedCall(1, 1, 0)
		if err != nil {
			return nil, err
		}

		// Load the result into map
		r := make(map[string]interface{})
		l.PushNil() // Add nil entry on stack (need 2 free slots).
		for l.Next(-2) {
			key, ok := l.ToString(-2)
			if !ok {
				return nil, fmt.Errorf("returned keys must be string")
			}
			val := l.ToValue(-1)
			l.Pop(1) // Remove val, but need key for the next iter.

			r[key] = val
		}

		return r, nil
	}, nil
}

func MakeParser(p string) (Parser, error) {
	var opts []string
	if i := strings.IndexByte(p, ':'); i != -1 {
		opts = strings.Split(p[i+1:], ":")
		p = p[0:i]
	}

	switch p {
	case "string":
		return MakeStringParser(opts)

	case "bool":
		return MakeBoolParser(opts)

	case "int":
		return MakeIntParser(opts)

	case "float":
		return MakeFloatParser(opts)

	case "lua":
		return MakeLuaParser(opts)

	default:
		return nil, fmt.Errorf("parser not supported: %s", p)
	}
}
