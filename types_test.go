// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package strum_test

import (
	"bytes"
	"fmt"
	"math"
	"math/bits"
	"testing"
	"time"

	"github.com/xdg-go/strum"
)

func TestDecodeBool(t *testing.T) {
	cases := []struct {
		label       string
		input       string
		want        bool
		errContains string
	}{
		{
			label: "false",
			input: "false",
			want:  false,
		},
		{
			label: "true",
			input: "true",
			want:  true,
		},
		{
			label: "mixed case true",
			input: "trUe",
			want:  true,
		},
		{
			label: "upper case false",
			input: "FALSE",
			want:  false,
		},
		{
			label:       "invalid string",
			input:       "yes",
			errContains: "error decoding",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got bool
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
	cases := []struct {
		label       string
		input       string
		want        time.Time
		errContains string
	}{
		{
			label: "RFC3339",
			input: "2021-01-01T00:00:00Z",
			want:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			label:       "invalid string",
			input:       "not-a-date-string",
			errContains: "cannot parse",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got time.Time
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}

func TestDecodeStruct(t *testing.T) {
	type person struct {
		Name   string
		Age    int
		Active bool
		Date   time.Time
	}

	cases := []struct {
		label       string
		input       string
		want        person
		errContains string
	}{
		{
			label: "",
			input: "John 42 true 2021-01-01T00:00:00Z",
			want:  person{"John", 42, true, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			var got person
			err := d.Decode(&got)
			errContains(t, err, c.errContains, "decode error")
			if err == nil {
				isWantGot(t, c.want, got, "decode result")
			}
		})
	}
}
