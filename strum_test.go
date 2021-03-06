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
	"math/big"
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

			// This is long-hand versus using DecodeAll to verify that it
			// actually works as expected line-by-line.
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

func TestDecodeSingleToken(t *testing.T) {
	cases := []testcase{
		{
			label: "int",
			input: "23",
			want:  func() interface{} { return 23 },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got int
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "no tokens",
			input: "\n",
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got int
				err := d.Decode(&got)
				return got, err
			},
			errContains: "decoding int: expected 1 token, but found 0",
		},
		{
			label: "two tokens",
			input: "42 23",
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got int
				err := d.Decode(&got)
				return got, err
			},
			errContains: "decoding int: expected 1 token, but found 2",
		},
		{
			label: "uint",
			input: "45",
			want:  func() interface{} { return uint(45) },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got uint
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "float32",
			input: "4.5",
			want:  func() interface{} { return float32(4.5) },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got float32
				err := d.Decode(&got)
				return got, err
			},
		},
	}

	testTestCases(t, cases)
}

func TestDecodeEntireLine(t *testing.T) {
	cases := []testcase{
		{
			label: "string",
			input: "hello world",
			want:  func() interface{} { return "hello world" },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got string
				err := d.Decode(&got)
				return got, err
			},
		},
	}

	testTestCases(t, cases)
}

func TestDecodeSlices(t *testing.T) {
	cases := []testcase{
		{
			label: "bool: false true",
			input: "false true",
			want:  func() interface{} { return []bool{false, true} },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []bool
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "bool",
			input: "false junk",
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []bool
				err := d.Decode(&got)
				return got, err
			},
			errContains: "error decoding",
		},
		{
			label: "string",
			input: "hello world",
			want:  func() interface{} { return []string{"hello", "world"} },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []string
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "int",
			input: "23 45",
			want:  func() interface{} { return []int{23, 45} },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []int
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "uint",
			input: "23 45",
			want:  func() interface{} { return []uint{23, 45} },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []uint
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "float64",
			input: "3.14 1.41",
			want:  func() interface{} { return []float64{3.14, 1.41} },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []float64
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "time.Time",
			input: "2021 2022",
			want: func() interface{} {
				return []time.Time{
					time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				}
			},
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []time.Time
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "time.Duration",
			input: "10s 10m",
			want: func() interface{} {
				return []time.Duration{
					time.Duration(10 * time.Second),
					time.Duration(10 * time.Minute),
				}
			},
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []time.Duration
				err := d.Decode(&got)
				return got, err
			},
		},
		{
			label: "big.Rat",
			input: "1/2 1/3 1/4",
			want: func() interface{} {
				return []*big.Rat{
					big.NewRat(1, 2),
					big.NewRat(1, 3),
					big.NewRat(1, 4),
				}
			},
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var got []*big.Rat
				err := d.Decode(&got)
				// Must stringify results for comparison
				return got, err
			},
			normalize: func(v interface{}) interface{} { return fmt.Sprintf("%v", v) },
		},
	}

	testTestCases(t, cases)
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

	type partPrivate struct {
		First  string
		second string
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

	// private fields
	{
		var pp []partPrivate
		err := strum.Unmarshal([]byte("hello world"), &pp)
		errContains(t, err, "cannot decode to unexported field", "private fields")
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
