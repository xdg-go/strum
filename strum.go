// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package strum provides a string unmarshaler to convert line-oriented text
// (such as from STDIN) into simple Go types:
//
//  - strings
//  - booleans ('true', 'false'; case insensitive)
//  - integers (signed and unsigned, all widths)
//  - floats (32-bit and 64-bit)
//
// For integers, all Go integer literal formats are supported, including
// base prefixes (`0xff`) and underscores (`1_000_000`).
//
// Additionally, there is special support for certain types:
//
//  - time.Time (only RFC 3339 strings supported at the moment)
//
// strum also supports decoding into structs, but the fields must be
// one of the supported simple types above. Recursive structs
// are not supported.
//
// Field extraction defaults to whitespace-separated fields, but strum
// supports using delimiters, regular expressions, or a custom tokenizer.
package strum

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
)

// A Tokenizer is a function that breaks an input string into tokens.
type Tokenizer func(s string) ([]string, error)

// A Decoder converts an input stream into structs.
type Decoder struct {
	s *bufio.Scanner
	t Tokenizer
}

// NewDecoder returns a Decoder that reads from r. The default Decoder will
// tokenize with `strings.Fields` function.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		s: bufio.NewScanner(r),
		t: func(s string) ([]string, error) { return strings.Fields(s), nil },
	}
}

// WithTokenizer modifies a Decoder to use a customer tokenizing function.
func (d *Decoder) WithTokenizer(t Tokenizer) *Decoder {
	d.t = t
	return d
}

// WithTokenRegexp modifies a Decoder to use a regular expression to extract
// string fields.  The regular expression is called with `FindStringSubmatches`
// for each line of input, so it must encompass an entire line of input.
func (d *Decoder) WithTokenRegexp(re *regexp.Regexp) *Decoder {
	return d.WithTokenizer(
		func(s string) ([]string, error) {
			xs := re.FindStringSubmatch(s)
			if xs == nil {
				return []string{}, errors.New("regexp failed to match line " + s)
			}
			// If the regexp had no submatches, then there are no tokens.
			// XXX Should this be an error?
			if len(xs) == 1 {
				return []string{}, nil
			}
			// Drop the full match and return only submatches.
			return xs[1:], nil
		},
	)
}

// WithSplitOn modifies a Decoder to split fields on a separator string.
func (d *Decoder) WithSplitOn(sep string) *Decoder {
	return d.WithTokenizer(
		func(s string) ([]string, error) {
			return strings.Split(s, sep), nil
		},
	)
}

// Tokens returns all strings generated by the tokenizer.  It is used
// internally by `Decode`, but made available for testing and diagnostics.
func (d *Decoder) Tokens() ([]string, error) {
	s, err := d.readline()
	if err != nil {
		return nil, err
	}
	return d.t(s)
}

func (d *Decoder) readline() (string, error) {
	if !(d.s.Scan()) {
		err := d.s.Err()
		if err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return d.s.Text(), nil
}

// Decode reads the next line of input, tokenizes it, and converts tokens
// into the fields of `v` in order.  The `v` argument must be a pointer to
// a struct.  If the input has fewer tokens than fields in the struct, the
// extra fields will be left with zero values.
func (d *Decoder) Decode(v interface{}) error {
	argValue := reflect.ValueOf(v)

	if argValue.Kind() != reflect.Ptr {
		return fmt.Errorf("argument to Decode must be a pointer, not %s", argValue.Kind())
	}

	// XXX What if pointer is nil?

	return d.decode(argValue.Elem())
}

func (d *Decoder) decode(destValue reflect.Value) error {
	// Handle certain types specially, not as their underlying data kind.
	switch destValue.Type() {
	case timeType:
		return d.decodeSingleToken("time.Time", destValue)
	}

	switch destValue.Kind() {
	case reflect.Bool:
		return d.decodeSingleToken("bool", destValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return d.decodeSingleToken("int", destValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return d.decodeSingleToken("uint", destValue)
	case reflect.Float32, reflect.Float64:
		return d.decodeSingleToken("float", destValue)
	case reflect.Struct:
		return d.decodeStruct(destValue)
	default:
		return fmt.Errorf("cannot Decode into pointer to %s", destValue.Kind())
	}
}

func (d *Decoder) decodeStruct(destValue reflect.Value) error {
	tokens, err := d.Tokens()
	if err != nil {
		return err
	}

	destType := destValue.Type()
	destNS := destType.PkgPath() + "." + destType.Name()

	// map tokens into argValue
	numFields := destValue.NumField()
	for i := range tokens {
		if i >= numFields {
			break
		}
		fieldName := destNS + "." + destType.Field(i).Name
		err = decodeToValue(fieldName, destValue.Field(i), tokens[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) decodeSingleToken(name string, destValue reflect.Value) error {
	tokens, err := d.Tokens()
	if err != nil {
		return err
	}

	if len(tokens) != 1 {
		return fmt.Errorf("decoding %s: expected 1 token, but found %d", name, len(tokens))
	}

	return decodeToValue(name, destValue, tokens[0])
}

func (d *Decoder) DecodeAll(v interface{}) error {
	argValue := reflect.ValueOf(v)

	// XXX What if Nil?
	if argValue.Kind() != reflect.Ptr {
		return fmt.Errorf("argument to DecodeAll must be a pointer, not %s", argValue.Kind())
	}

	sliceValue := argValue.Elem()
	if sliceValue.Kind() != reflect.Slice {
		return fmt.Errorf("argument to DecodeAll must be a pointer to slice of struct, not %s", sliceValue.Kind())
	}

	sliceType := sliceValue.Type()

	// Make a zero-length slice if it starts uninitialized
	if sliceValue.IsNil() {
		sliceValue.Set(reflect.MakeSlice(sliceType, 0, 1))
	}

	// Decode every line into the slice
	for {
		v := reflect.New(sliceType.Elem()).Elem()
		err := d.decode(v)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, v))
	}
}
