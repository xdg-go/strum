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
func isDecodableValue(v reflect.Value) bool {
	switch v.Type() {
	case durationType:
		return true
	case timeType, timePtrType:
		return true
	}

	if isTextUnmarshaler(v) {
		return true
	}

	switch v.Kind() {
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

func isTextUnmarshaler(v reflect.Value) bool {
	return v.Type().Implements(textUnmarshalerType)
}

func (d *Decoder) decodeToValue(name string, v reflect.Value, s string) error {
	// Custom parsing for certain types
	switch v.Type() {
	case durationType:
		t, err := time.ParseDuration(s)
		if err != nil {
			return decodingError(name, err)
		}
		v.Set(reflect.ValueOf(t))
		return nil
	case timeType:
		t, err := d.dp(s)
		if err != nil {
			return decodingError(name, err)
		}
		v.Set(reflect.ValueOf(t))
		return nil
	case timePtrType:
		// Handle recursively to avoid using TextUnmarshaler
		maybeInstantiatePtr(v)
		return d.decodeToValue(name, v.Elem(), s)
	}

	// Handle TextUnmarshaler types
	if isTextUnmarshaler(v) {
		maybeInstantiatePtr(v)
		f := v.MethodByName("UnmarshalText")
		xs := []byte(s)
		args := []reflect.Value{reflect.ValueOf(xs)}
		ret := f.Call(args)
		if !ret[0].IsNil() {
			return decodingError(name, ret[0].Interface().(error))
		}
		return nil
	}

	switch v.Kind() {
	case reflect.Bool:
		switch strings.ToLower(s) {
		case "true":
			v.SetBool(true)
		case "false":
			v.SetBool(false)
		default:
			return decodingError(name, fmt.Errorf("error decoding '%s' as boolean", s))
		}
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 0, v.Type().Bits())
		if err != nil {
			return decodingError(name, err)
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 0, v.Type().Bits())
		if err != nil {
			return decodingError(name, err)
		}
		v.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return decodingError(name, err)
		}
		v.SetFloat(f)
	case reflect.Ptr:
		maybeInstantiatePtr(v)
		return d.decodeToValue(name, v.Elem(), s)
	default:
		return decodingError(name, fmt.Errorf("unsupported type %s", v.Type()))
	}

	return nil
}

func maybeInstantiatePtr(v reflect.Value) {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		np := reflect.New(v.Type().Elem())
		v.Set(np)
	}
}
