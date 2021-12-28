// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package strum_test

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/xdg-go/strum"
)

func isWantGot(t *testing.T, want, got interface{}, label string) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", label, diff)
	}
}

func errContains(t *testing.T, err error, contains string, label string) {
	t.Helper()
	if err == nil {
		if contains == "" {
			return
		}
		t.Errorf("%s expected error containing '%s', but got no error", label, contains)
	} else {
		if contains == "" {
			t.Errorf("%s expected no error, but got '%s'", label, err.Error())
		}
		if !strings.Contains(err.Error(), contains) {
			t.Errorf("%s expected error containing '%s', but got '%s'", label, contains, err.Error())
		}
	}
}

func TestSplitDefault(t *testing.T) {
	text := "John Doe 2021-01-01\n"
	expect := []string{"John", "Doe", "2021-01-01"}

	r := bytes.NewBufferString(text)

	d := strum.NewDecoder(r)

	words, err := d.Tokens()
	if err != nil {
		t.Error(err)
	}

	isWantGot(t, expect, words, "tokenizing on whitespace")
}

func TestSplitOn(t *testing.T) {
	text := "John Doe|2021-01-01\n"
	expect := []string{"John Doe", "2021-01-01"}

	r := bytes.NewBufferString(text)

	d := strum.NewDecoder(r).WithSplitOn("|")

	words, err := d.Tokens()
	if err != nil {
		t.Error(err)
	}

	isWantGot(t, expect, words, "tokenizing WithSpliton")
}

func TestRegexp(t *testing.T) {
	text := "John 23x42\n"
	expect := []string{"John", "23", "42"}

	r := bytes.NewBufferString(text)

	re := regexp.MustCompile(`^(\S+)\s+(\d+)x(\d+)`)
	d := strum.NewDecoder(r).WithTokenRegexp(re)

	words, err := d.Tokens()
	if err != nil {
		t.Error(err)
	}

	isWantGot(t, expect, words, "tokenizing with regexp")
}

func TestRegexpBad(t *testing.T) {
	text := "John 23x42\n"

	r := bytes.NewBufferString(text)

	re := regexp.MustCompile(`^.*$`)
	d := strum.NewDecoder(r).WithTokenRegexp(re)

	_, err := d.Tokens()
	errContains(t, err, "regexp has no subexpressions", "bad regexp")
}

func TestRegexpNoMatch(t *testing.T) {
	text := "John 23x42\n"

	r := bytes.NewBufferString(text)

	re := regexp.MustCompile(`^(\d+)`)
	d := strum.NewDecoder(r).WithTokenRegexp(re)

	_, err := d.Tokens()
	errContains(t, err, "regexp failed to match line", "regexp didn't match")
}

func TestDateParser(t *testing.T) {
	text := "not-a-date"
	r := bytes.NewBufferString(text)
	want := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	d := strum.NewDecoder(r).WithDateParser(func(s string) (time.Time, error) {
		return want, nil
	})
	var got time.Time
	err := d.Decode(&got)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, want, got, "custom date parser")
}

func TestDecode(t *testing.T) {
	type person struct {
		Name   string
		Age    int
		Active bool
		Date   time.Time
	}

	lines := []string{
		"John 42 true 2021-01-01T00:00:00Z",
		"Jane 23 false 2022-01-01T00:00:00Z",
		"Jack 36 TrUe 2023-01-01T00:00:00Z",
	}

	expect := []person{
		{"John", 42, true, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Jane", 23, false, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Jack", 36, true, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	// Test with and without trailing newline
	trailingChar := []string{"\n", ""}

	for _, ch := range trailingChar {
		ch := ch
		label := "trailing newline"
		if ch == "" {
			label = "no trailing newline"
		}
		t.Run(label, func(t *testing.T) {
			r := bytes.NewBufferString(strings.Join(lines, "\n") + ch)
			d := strum.NewDecoder(r)

			output := []person{}

			for {
				var p person
				err := d.Decode(&p)
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatal(err)
				}
				output = append(output, p)
			}

			if len(expect) != len(output) {
				t.Fatalf("Expected %d records, but got only %d", len(expect), len(output))
			}

			for i := range expect {
				isWantGot(t, expect[i], output[i], fmt.Sprintf("decoded record %d", i))
			}
		})
	}
}

func TestDecodeIntContainers(t *testing.T) {
	lines := []string{
		"42",
		"23",
		"36",
		"81",
	}

	ints := []int{42, 23, 36, 81}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var i int
	var output []int

	err := d.Decode(&i)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, ints[0], i, "Decode to int reference")

	err = d.DecodeAll(&output)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, ints[1:], output, "Decode to int slice")
}

func TestDecodeIntContainersErrors(t *testing.T) {
	cases := []struct {
		label       string
		input       string
		errContains string
	}{
		{
			label:       "no tokens",
			input:       "\n",
			errContains: "decoding int: expected 1 token, but found 0",
		},
		{
			label:       "two tokens",
			input:       "42 23",
			errContains: "decoding int: expected 1 token, but found 2",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var i int
			err := d.Decode(&i)
			errContains(t, err, c.errContains, "Decode error")
		})
	}
}

func TestDecodeUintContainers(t *testing.T) {
	lines := []string{
		"42",
		"23",
		"36",
		"81",
	}

	uints := []uint8{42, 23, 36, 81}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var u uint8
	var output []uint8

	err := d.Decode(&u)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, uints[0], u, "Decode to uint reference")

	err = d.DecodeAll(&output)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, uints[1:], output, "Decode to uint slice")
}

func TestDecodeFloatContainers(t *testing.T) {
	lines := []string{
		"4.2",
		"2.3",
		"3.6",
		"8.1",
	}

	floats := []float64{4.2, 2.3, 3.6, 8.1}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var u float64
	var output []float64

	err := d.Decode(&u)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, floats[0], u, "Decode to float reference")

	err = d.DecodeAll(&output)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, floats[1:], output, "Decode to float slice")
}

func TestDecodeBoolContainers(t *testing.T) {
	lines := []string{
		"42",
		"23",
		"36",
		"81",
	}

	uints := []uint8{42, 23, 36, 81}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var u uint8
	var output []uint8

	err := d.Decode(&u)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, uints[0], u, "Decode to uint reference")

	err = d.DecodeAll(&output)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, uints[1:], output, "Decode to int slice")
}

func TestDecodeStringContainers(t *testing.T) {
	lines := []string{
		"42",
		"23",
		"36",
		"81",
	}

	xs := []string{"42", "23", "36", "81"}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var s string
	var output []string

	err := d.Decode(&s)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, xs[0], s, "Decode to string reference")

	err = d.DecodeAll(&output)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, xs[1:], output, "Decode to string slice")
}

func TestDecodeSliceOfBool(t *testing.T) {
	cases := []struct {
		label       string
		input       string
		want        []bool
		errContains string
	}{
		{
			label: "false true",
			input: "false true",
			want:  []bool{false, true},
		},
		{
			label:       "false junk",
			input:       "false junk",
			errContains: "error decoding",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got []bool
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}

func TestDecodeSliceOfString(t *testing.T) {
	cases := []struct {
		label       string
		input       string
		want        []string
		errContains string
	}{
		{
			label: "false true",
			input: "false true",
			want:  []string{"false", "true"},
		},
		{
			label: "false junk",
			input: "false junk",
			want:  []string{"false", "junk"},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got []string
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}

func TestDecodeAll(t *testing.T) {
	type person struct {
		Name   string
		Age    int
		Active bool
		Date   time.Time
	}

	lines := []string{
		"John 42 true 2021-01-01T00:00:00Z",
		"Jane 23 false 2022-01-01T00:00:00Z",
		"Jack 36 TrUe 2023-01-01T00:00:00Z",
	}

	expect := []person{
		{"John", 42, true, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Jane", 23, false, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Jack", 36, true, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)
	var output []person

	err := d.DecodeAll(&output)
	if err != nil {
		t.Fatalf("calling DecodeAll: %v", err)
	}

	isWantGot(t, len(expect), len(output), "length of decoded slice")

	for i := range expect {
		isWantGot(t, expect[i], output[i], fmt.Sprintf("decoded record %d", i))
	}
}

func TestBadTargets(t *testing.T) {
	type person struct {
		Name   string
		Age    int
		Active bool
		Date   time.Time
	}

	lines := []string{
		"John 42 true 2021-01-01T00:00:00Z",
		"Jane 23 false 2022-01-01T00:00:00Z",
		"Jack 36 TrUe 2023-01-01T00:00:00Z",
	}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	// non-pointer
	{
		var v person
		err := d.Decode(v)
		errContains(t, err, "argument must be a pointer", "Decode with non-pointer")

		var output []person
		err = d.DecodeAll(output)
		errContains(t, err, "argument must be a pointer", "DecodeAll with non-pointer")
	}

	// pointer to invalid types
	{
		var v complex128
		err := d.Decode(&v)
		errContains(t, err, "cannot decode into type complex128", "Decode with pointer to unsupported type")

		var output map[string]string
		err = d.DecodeAll(&output)
		errContains(t, err, "argument must be a pointer to slice, not", "DecodeAll with pointer to slice of unsupported type")
	}

	// nil
	{
		err := d.Decode(nil)
		errContains(t, err, "argument must be a non-nil pointer", "Decode literal nil")

		var vp *string
		err = d.Decode(vp)
		errContains(t, err, "argument must be a non-nil pointer", "Decode nil pointer")

		err = d.DecodeAll(nil)
		errContains(t, err, "argument must be a non-nil pointer", "DecodeAll literal nil")

		var xs *[]string
		err = d.DecodeAll(xs)
		errContains(t, err, "argument must be a non-nil pointer", "DecodeAll nil pointer")
	}

	// slice of struct
	{
		var sp []person
		err := d.Decode(&sp)
		errContains(t, err, "decoding to this slice type not supported: []strum_test.person", "slice of struct")
	}

	// unmarshal
	{
		var i int
		err := strum.Unmarshal([]byte("hello"), &i)
		errContains(t, err, "argument must be a pointer to slice", "Unmarshal")
	}
}

func TestUnmarshal(t *testing.T) {
	input := "hello world\ngoodbye world"
	want := []string{
		"hello world",
		"goodbye world",
	}
	var xs []string
	err := strum.Unmarshal([]byte(input), &xs)
	if err != nil {
		t.Fatal(err)
	}
	isWantGot(t, want, xs, "unmarshal string")
}
