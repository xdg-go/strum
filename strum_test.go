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
	"math"
	"math/bits"
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

func TestDecodeBool(t *testing.T) {
	type boolean struct {
		B bool
	}

	cases := []struct {
		label       string
		input       string
		want        boolean
		errContains string
	}{
		{
			label: "false",
			input: "false",
			want:  boolean{false},
		},
		{
			label: "true",
			input: "true",
			want:  boolean{true},
		},
		{
			label: "mixed case true",
			input: "trUe",
			want:  boolean{true},
		},
		{
			label: "upper case false",
			input: "FALSE",
			want:  boolean{false},
		},
		{
			label:       "invalid string",
			input:       "yes",
			want:        boolean{},
			errContains: "error decoding",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got boolean
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}

func TestDecodeInts(t *testing.T) {
	type ints struct {
		I   int
		I8  int8
		I16 int16
		I36 int32
		I64 int64
	}

	maxInt := 1<<(bits.UintSize-1) - 1
	minInt := -1 << (bits.UintSize - 1)

	maxIntStr := fmt.Sprintf("%d", maxInt)
	minIntStr := fmt.Sprintf("%d", minInt)

	cases := []struct {
		label       string
		input       string
		want        ints
		errContains string
	}{
		{
			label: "all zeros",
			input: "0 0 0 0 0",
			want:  ints{},
		},
		{
			label: "positive decimal",
			input: "1 2 3 4 5",
			want:  ints{1, 2, 3, 4, 5},
		},
		{
			label: "negative decimal",
			input: "-1 -2 -3 -4 -5",
			want:  ints{-1, -2, -3, -4, -5},
		},
		{
			label: "positive hex",
			input: "0xa 0xb 0xc 0xd 0xe",
			want:  ints{10, 11, 12, 13, 14},
		},
		{
			label: "negative hex",
			input: "-0xa -0xb -0xc -0xd -0xe",
			want:  ints{-10, -11, -12, -13, -14},
		},
		{
			label: "maxints",
			input: maxIntStr + " 127 32767 2_147_483_647 9_223_372_036_854_775_807",
			want:  ints{maxInt, math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64},
		},
		{
			label: "minints",
			input: minIntStr + " -128 -32768 -2_147_483_648 -9_223_372_036_854_775_808",
			want:  ints{minInt, math.MinInt8, math.MinInt16, math.MinInt32, math.MinInt64},
		},
		{
			label:       "overflow maxint8",
			input:       "1 128 3 4 5",
			want:        ints{},
			errContains: "out of range",
		},
		{
			label:       "overflow minint8",
			input:       "1 -129 3 4 5",
			want:        ints{},
			errContains: "out of range",
		},
		{
			label:       "overflow maxint64",
			input:       "1 127 32767 2_147_483_647 9_223_372_036_854_775_808",
			want:        ints{},
			errContains: "out of range",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got ints
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}

func TestDecodeUints(t *testing.T) {
	type uints struct {
		U   uint
		U8  uint8
		U16 uint16
		U36 uint32
		U64 uint64
	}

	maxUint := ^uint(0)
	maxUintStr := fmt.Sprintf("%d", maxUint)

	cases := []struct {
		label       string
		input       string
		want        uints
		errContains string
	}{
		{
			label: "all zeros",
			input: "0 0 0 0 0",
			want:  uints{},
		},
		{
			label: "positive decimal",
			input: "1 2 3 4 5",
			want:  uints{1, 2, 3, 4, 5},
		},
		{
			label:       "negative decimal",
			input:       "-1 -2 -3 -4 -5",
			errContains: "invalid syntax",
		},
		{
			label: "positive hex",
			input: "0xa 0xb 0xc 0xd 0xe",
			want:  uints{10, 11, 12, 13, 14},
		},
		{
			label: "maxuints",
			input: maxUintStr + " 255 65535 4_294_967_295 18_446_744_073_709_551_615",
			want:  uints{maxUint, math.MaxUint8, math.MaxUint16, math.MaxUint32, math.MaxUint64},
		},
		{
			label:       "overflow maxuint8",
			input:       "1 256 3 4 5",
			want:        uints{},
			errContains: "out of range",
		},
		{
			label:       "overflow maxuint64",
			input:       "1 127 32767 2_147_483_647 18_446_744_073_709_551_616",
			want:        uints{},
			errContains: "out of range",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got uints
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}

func TestDecodeDate(t *testing.T) {
	type dates struct {
		D time.Time
	}

	cases := []struct {
		label       string
		input       string
		want        dates
		errContains string
	}{
		{
			label: "RFC3339",
			input: "2021-01-01T00:00:00Z",
			want:  dates{time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		{
			label:       "invalid string",
			input:       "not-a-date-string",
			want:        dates{},
			errContains: "cannot parse",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got dates
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}
