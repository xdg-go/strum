// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package strum provides a string unmarshaler to tokenize line-oriented text
// (such as from stdin) and convert tokens into simple Go types.
//
// Tokenization defaults to whitespace-separated fields, but strum supports
// using delimiters, regular expressions, or a custom tokenizer.
//
// A line with a single token can be unmarshaled into a single variable of any
// supported type.
//
// A line with multiple tokens can be unmarshaled into a slice or a struct of
// supported types.  It can also be unmarshaled into a single string, in which
// case tokenization is skipped.
//
// Trying to unmarshal multiple tokens into a single variable or too many tokens
// for the number of fields in a struct will result in an error.  Having too few
// tokens for the fields in a struct is allowed; remaining fields will be
// zeroed.  When unmarshaling to a slice, decoded values are appended; existing
// values are untouched.
//
// strum supports the following types:
//
//  - strings
//  - booleans ('true', 'false'; case insensitive)
//  - integers (signed and unsigned, all widths)
//  - floats (32-bit and 64-bit)
//
// Additionally, there is special support for certain types:
//
//  - time.Duration
//  - time.Time
//  - any type implementing encoding.TextUnmarshaler
//  - pointers to supported types (which will auto-instantiate)
//
// For numeric types, all Go literal formats are supported, including base
// prefixes (`0xff`) and underscores (`1_000_000`) for integers.
//
// For time.Time, strum detects and parses  a wide varity of formats using the
// github.com/araddon/dateparse library. By default, it favors United States
// interpretation of MM/DD/YYYY and has time zone semantics equivalent to
// `time.Parse`.  strum allows specifying a custom parser instead.
//
// strum provides `DecodeAll` to unmarshal all lines of input at once.
package strum

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

// A Tokenizer is a function that breaks an input string into tokens.
type Tokenizer func(s string) ([]string, error)

// A DateParser parses a string into a time.Time struct.
type DateParser func(s string) (time.Time, error)

// A Decoder converts an input stream into Go types.
type Decoder struct {
	s  *bufio.Scanner
	t  Tokenizer
	dp DateParser
}

// NewDecoder returns a Decoder that reads from r. The default Decoder will
// tokenize with `strings.Fields` function. The default date parser uses
// github.com/araddon/dateparse.ParseAny.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		s:  bufio.NewScanner(r),
		t:  func(s string) ([]string, error) { return strings.Fields(s), nil },
		dp: func(s string) (time.Time, error) { return dateparse.ParseAny(s) },
	}
}

// WithDateParser modifies a Decoder to use a custom date parsing function.
func (d *Decoder) WithDateParser(dp DateParser) *Decoder {
	d.dp = dp
	return d
}

// WithTokenizer modifies a Decoder to use a custom tokenizing function.
func (d *Decoder) WithTokenizer(t Tokenizer) *Decoder {
	d.t = t
	return d
}

// WithTokenRegexp modifies a Decoder to use a regular expression to extract
// tokens.  The regular expression is called with `FindStringSubmatches` for
// each line of input, so it must encompass an entire line of input.  If the
// line fails to match or if the regular expression has no subexpressions, an
// error is returned.
func (d *Decoder) WithTokenRegexp(re *regexp.Regexp) *Decoder {
	return d.WithTokenizer(
		func(s string) ([]string, error) {
			xs := re.FindStringSubmatch(s)
			if xs == nil {
				return []string{}, errors.New("regexp failed to match line " + s)
			}
			// A regexp without capture expressions is an error.
			if len(xs) == 1 {
				return []string{}, errors.New("regexp has no subexpressions")
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

// Tokens consumes a line of input and returns all strings generated by the
// tokenizer.  It is used internally by `Decode`, but available for testing or
// for skipping over a line of input that should not be decoded.
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

// Decode reads the next line of input and stores it in the value pointed to by
// `v`. It returns `io.EOF` when no more data is available.
func (d *Decoder) Decode(v interface{}) error {
	destValue, err := extractDestValue(v)
	if err != nil {
		return fmt.Errorf("Decode: %w", err)
	}
	return d.decode(destValue)
}

// decode puts a single line of input into a destination. It invokes a type-aware,
// decoding routine that determines whether the line must have a single token,
// or be consumed as a line, or whether multiple tokens are decoded to a slice
// or struct.  It also recursively dereferences pointers to find an element to
// decode in case they are pointers to structs, slices, or text unmarshalers.
func (d *Decoder) decode(destValue reflect.Value) error {
	// Handle certain types specially, not as their underlying data kind.
	switch destValue.Type() {
	case durationType:
		return d.decodeSingleToken(destValue)
	case timeType, timePtrType:
		return d.decodeSingleToken(destValue)
	}

	// Handle text unmarshaler types
	if isTextUnmarshaler(destValue) {
		return d.decodeSingleToken(destValue)
	}

	switch destValue.Kind() {
	case reflect.Bool:
		return d.decodeSingleToken(destValue)
	case reflect.String:
		return d.decodeLine(destValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return d.decodeSingleToken(destValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return d.decodeSingleToken(destValue)
	case reflect.Float32, reflect.Float64:
		return d.decodeSingleToken(destValue)
	case reflect.Struct:
		return d.decodeStruct(destValue)
	case reflect.Slice:
		return d.decodeSlice(destValue)
	case reflect.Ptr:
		maybeInstantiatePtr(destValue)
		return d.decode(destValue.Elem())
	default:
		return fmt.Errorf("cannot decode into type %s", destValue.Type())
	}
}

func (d *Decoder) decodeStruct(destValue reflect.Value) error {
	tokens, err := d.Tokens()
	if err != nil {
		return err
	}

	destType := destValue.Type()

	// Zero the struct so any prior fields are reset.
	destValue.Set(reflect.New(destType).Elem())

	// Map tokens into argValue
	numFields := destValue.NumField()
	for i := range tokens {
		if i >= numFields {
			return fmt.Errorf("too many tokens for struct %s", destValue.Type())
		}
		fieldName := destType.Name() + "." + destType.Field(i).Name
		// PkgPath is empty for exported fields.  See https://pkg.go.dev/reflect#StructField
		// In Go 1.17, this is available as `IsExported`.
		if destType.Field(i).PkgPath != "" {
			return fmt.Errorf("cannot decode to unexported field %s", fieldName)
		}
		err = d.decodeToValue(fieldName, destValue.Field(i), tokens[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) decodeSlice(sliceValue reflect.Value) error {
	if !isDecodableValue(reflect.New(sliceValue.Type().Elem()).Elem()) {
		return fmt.Errorf("decoding to this slice type not supported: %s", sliceValue.Type())
	}

	tokens, err := d.Tokens()
	if err != nil {
		return err
	}

	sliceType := sliceValue.Type()

	for i, s := range tokens {
		v := reflect.New(sliceType.Elem()).Elem()
		err := d.decodeToValue(fmt.Sprintf("element %d", i), v, s)
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, v))
	}

	return nil
}

func (d *Decoder) decodeSingleToken(destValue reflect.Value) error {
	tokens, err := d.Tokens()
	if err != nil {
		return err
	}

	if len(tokens) != 1 {
		return fmt.Errorf("decoding %s: expected 1 token, but found %d", destValue.Type(), len(tokens))
	}

	return d.decodeToValue(destValue.Type().String(), destValue, tokens[0])
}

func (d *Decoder) decodeLine(destValue reflect.Value) error {
	line, err := d.readline()
	if err != nil {
		return err
	}

	return d.decodeToValue(destValue.Type().String(), destValue, line)
}

// DecodeAll reads the remaining lines of input into `v`, where `v` must be a
// pointer to a slice of a type that would valid for Decode.  It works as if
// `Decode` were called for all lines and the resulting values were appended to
// the slice.  If `v` points to an uninitialized slice, the slice will be
// created. DecodeAll returns `nil` when EOF is reached.
func (d *Decoder) DecodeAll(v interface{}) error {
	sliceValue, err := extractDestSlice(v)
	if err != nil {
		return fmt.Errorf("DecodeAll: %w", err)
	}
	return d.decodeAll(sliceValue)
}

func (d *Decoder) decodeAll(sliceValue reflect.Value) error {
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

// Unmarshal parses the input data as newline delimited strings and appends the
// result to the value pointed to by `v`, where `v` must be a pointer to a slice
// of a type that would valid for Decode.  If `v` points to an uninitialized
// slice, the slice will be created.
func Unmarshal(data []byte, v interface{}) error {
	sliceValue, err := extractDestSlice(v)
	if err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	r := bytes.NewBuffer(data)
	d := NewDecoder(r)
	return d.decodeAll(sliceValue)
}

func extractDestValue(v interface{}) (reflect.Value, error) {
	if v == nil {
		return reflect.Value{}, fmt.Errorf("argument must be a non-nil pointer")
	}

	argValue := reflect.ValueOf(v)

	if argValue.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("argument must be a pointer, not %s", argValue.Kind())
	}

	if argValue.IsNil() {
		return reflect.Value{}, fmt.Errorf("argument must be a non-nil pointer")
	}

	return argValue.Elem(), nil
}

func extractDestSlice(v interface{}) (reflect.Value, error) {
	sliceValue, err := extractDestValue(v)
	if err != nil {
		return reflect.Value{}, err
	}

	if sliceValue.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("argument must be a pointer to slice, not %s", sliceValue.Kind())
	}

	return sliceValue, nil
}
