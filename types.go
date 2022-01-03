// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package strum

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func decodingError(name string, err error) error {
	return fmt.Errorf("error decoding to %s: %w", name, err)
}

var durationType = reflect.TypeOf(time.Duration(0))
var timeType = reflect.TypeOf(time.Time{})
var timePtrType = reflect.TypeOf(&time.Time{})

// isDecodableValue duplicates the logic tree of `decodeToValue` to allow input
// validation before decoding is called. This supports better error messages.
func isDecodableValue(rv reflect.Value) bool {
	switch rv.Type() {
	case durationType:
		return true
	case timeType, timePtrType:
		return true
	}

	if isTextUnmarshaler(rv) {
		return true
	}

	switch rv.Kind() {
	case reflect.Bool:
		return true
	case reflect.String:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func isTextUnmarshaler(rv reflect.Value) bool {
	return rv.Type().Implements(textUnmarshalerType)
}

func (d *Decoder) decodeToValue(name string, rv reflect.Value, s string) error {
	// Custom parsing for certain types
	switch rv.Type() {
	case durationType:
		t, err := time.ParseDuration(s)
		if err != nil {
			return decodingError(name, err)
		}
		rv.Set(reflect.ValueOf(t))
		return nil
	case timeType:
		t, err := d.dp(s)
		if err != nil {
			return decodingError(name, err)
		}
		rv.Set(reflect.ValueOf(t))
		return nil
	case timePtrType:
		// Handle recursively to avoid using TextUnmarshaler
		maybeInstantiatePtr(rv)
		return d.decodeToValue(name, rv.Elem(), s)
	}

	// Handle TextUnmarshaler types
	if isTextUnmarshaler(rv) {
		maybeInstantiatePtr(rv)
		f := rv.MethodByName("UnmarshalText")
		xs := []byte(s)
		args := []reflect.Value{reflect.ValueOf(xs)}
		ret := f.Call(args)
		if !ret[0].IsNil() {
			return decodingError(name, ret[0].Interface().(error))
		}
		return nil
	}

	switch rv.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.ToLower(s))
		if err != nil {
			return decodingError(name, err)
		}
		rv.SetBool(b)
	case reflect.String:
		rv.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 0, rv.Type().Bits())
		if err != nil {
			return decodingError(name, err)
		}
		rv.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 0, rv.Type().Bits())
		if err != nil {
			return decodingError(name, err)
		}
		rv.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, rv.Type().Bits())
		if err != nil {
			return decodingError(name, err)
		}
		rv.SetFloat(f)
	case reflect.Ptr:
		maybeInstantiatePtr(rv)
		return d.decodeToValue(name, rv.Elem(), s)
	default:
		return decodingError(name, fmt.Errorf("unsupported type %s", rv.Type()))
	}

	return nil
}

func maybeInstantiatePtr(rv reflect.Value) {
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		np := reflect.New(rv.Type().Elem())
		rv.Set(np)
	}
}
