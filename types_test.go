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
	"math/big"
	"math/bits"
	"testing"
	"time"

	"github.com/xdg-go/strum"
)

type testcase struct {
	label       string
	input       string
	want        func() interface{}
	decode      func(*testing.T, *strum.Decoder) (interface{}, error)
	errContains string
	normalize   func(v interface{}) interface{}
}

func testTestCases(t *testing.T, cases []testcase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			r := bytes.NewBufferString(c.input)
			d := strum.NewDecoder(r)
			got, err := c.decode(t, d)
			if len(c.errContains) != 0 {
				errContains(t, err, c.errContains, "decoding")
				return
			}
			if err != nil {
				t.Fatalf("decoding: %v", err)
			}
			want := c.want()
			if c.normalize != nil {
				want = c.normalize(want)
				got = c.normalize(got)
			}
			isWantGot(t, want, got, "decode result")
		})
	}
}

func TestDecodeBool(t *testing.T) {
	boolDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got bool
		err := d.Decode(&got)
		return got, err
	}

	cases := []testcase{
		{
			label:  "false",
			input:  "false",
			want:   func() interface{} { return false },
			decode: boolDecode,
		},
		{
			label:  "true",
			input:  "true",
			want:   func() interface{} { return true },
			decode: boolDecode,
		},
		{
			label:  "mixed case true",
			input:  "trUe",
			want:   func() interface{} { return true },
			decode: boolDecode,
		},
		{
			label:  "upper case false",
			input:  "FALSE",
			want:   func() interface{} { return false },
			decode: boolDecode,
		},
		{
			label:       "invalid string",
			input:       "yes",
			decode:      boolDecode,
			errContains: "error decoding",
		},
	}

	testTestCases(t, cases)
}

func TestDecodeString(t *testing.T) {
	stringDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got string
		err := d.Decode(&got)
		return got, err
	}

	cases := []testcase{
		{
			label:  "a",
			input:  "a",
			want:   func() interface{} { return "a" },
			decode: stringDecode,
		},
		{
			label:  "a b",
			input:  "a b",
			want:   func() interface{} { return "a b" },
			decode: stringDecode,
		},
	}

	testTestCases(t, cases)
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

	intDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got ints
		err := d.Decode(&got)
		return got, err
	}
	cases := []testcase{
		{
			label:  "all zeros",
			input:  "0 0 0 0 0",
			want:   func() interface{} { return ints{} },
			decode: intDecode,
		},
		{
			label:  "positive decimal",
			input:  "1 2 3 4 5",
			want:   func() interface{} { return ints{1, 2, 3, 4, 5} },
			decode: intDecode,
		},
		{
			label:  "negative decimal",
			input:  "-1 -2 -3 -4 -5",
			want:   func() interface{} { return ints{-1, -2, -3, -4, -5} },
			decode: intDecode,
		},
		{
			label:  "positive hex",
			input:  "0xa 0xb 0xc 0xd 0xe",
			want:   func() interface{} { return ints{10, 11, 12, 13, 14} },
			decode: intDecode,
		},
		{
			label:  "negative hex",
			input:  "-0xa -0xb -0xc -0xd -0xe",
			want:   func() interface{} { return ints{-10, -11, -12, -13, -14} },
			decode: intDecode,
		},
		{
			label:  "maxints",
			input:  maxIntStr + " 127 32767 2_147_483_647 9_223_372_036_854_775_807",
			want:   func() interface{} { return ints{maxInt, math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64} },
			decode: intDecode,
		},
		{
			label:  "minints",
			input:  minIntStr + " -128 -32768 -2_147_483_648 -9_223_372_036_854_775_808",
			want:   func() interface{} { return ints{minInt, math.MinInt8, math.MinInt16, math.MinInt32, math.MinInt64} },
			decode: intDecode,
		},
		{
			label:       "overflow maxint8",
			input:       "1 128 3 4 5",
			decode:      intDecode,
			errContains: "out of range",
		},
		{
			label:       "overflow minint8",
			input:       "1 -129 3 4 5",
			decode:      intDecode,
			errContains: "out of range",
		},
		{
			label:       "overflow maxint64",
			input:       "1 127 32767 2_147_483_647 9_223_372_036_854_775_808",
			decode:      intDecode,
			errContains: "out of range",
		},
	}

	testTestCases(t, cases)
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

	uintDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got uints
		err := d.Decode(&got)
		return got, err
	}
	cases := []testcase{
		{
			label:  "all zeros",
			input:  "0 0 0 0 0",
			want:   func() interface{} { return uints{} },
			decode: uintDecode,
		},
		{
			label:  "positive decimal",
			input:  "1 2 3 4 5",
			want:   func() interface{} { return uints{1, 2, 3, 4, 5} },
			decode: uintDecode,
		},
		{
			label:       "negative decimal",
			input:       "-1 -2 -3 -4 -5",
			decode:      uintDecode,
			errContains: "invalid syntax",
		},
		{
			label:  "positive hex",
			input:  "0xa 0xb 0xc 0xd 0xe",
			decode: uintDecode,
			want:   func() interface{} { return uints{10, 11, 12, 13, 14} },
		},
		{
			label: "maxuints",
			input: maxUintStr + " 255 65535 4_294_967_295 18_446_744_073_709_551_615",
			want: func() interface{} {
				return uints{maxUint, math.MaxUint8, math.MaxUint16, math.MaxUint32, math.MaxUint64}
			},
			decode: uintDecode,
		},
		{
			label:       "overflow maxuint8",
			input:       "1 256 3 4 5",
			decode:      uintDecode,
			errContains: "out of range",
		},
		{
			label:       "overflow maxuint64",
			input:       "1 127 32767 2_147_483_647 18_446_744_073_709_551_616",
			decode:      uintDecode,
			errContains: "out of range",
		},
	}

	testTestCases(t, cases)
}

func TestDecodeFloats(t *testing.T) {
	type floats struct {
		F32 float32
		F64 float64
	}

	floatDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got floats
		err := d.Decode(&got)
		return got, err
	}
	cases := []testcase{
		{
			label:  "all zeros",
			input:  "0 0",
			want:   func() interface{} { return floats{} },
			decode: floatDecode,
		},
		{
			label:  "positive decimal",
			input:  "1.2 2.3",
			want:   func() interface{} { return floats{1.2, 2.3} },
			decode: floatDecode,
		},
		{
			label:  "negative decimal",
			input:  "-1.2 -2.3",
			want:   func() interface{} { return floats{-1.2, -2.3} },
			decode: floatDecode,
		},
		{
			label:  "specials",
			input:  "-Inf Inf",
			want:   func() interface{} { return floats{float32(math.Inf(-1)), math.Inf(1)} },
			decode: floatDecode,
		},
		{
			label:  "maxfloats",
			input:  "3.40282346638528859811704183484516925440e+38 1.79769313486231570814527423731704356798070e+308",
			want:   func() interface{} { return floats{math.MaxFloat32, math.MaxFloat64} },
			decode: floatDecode,
		},
		{
			label:  "small non-zero floats",
			input:  "1.401298464324817070923729583289916131280e-45 4.9406564584124654417656879286822137236505980e-324",
			want:   func() interface{} { return floats{math.SmallestNonzeroFloat32, math.SmallestNonzeroFloat64} },
			decode: floatDecode,
		},
		{
			label:       "out of range float32",
			input:       "1.0 3.41e310",
			decode:      floatDecode,
			errContains: "value out of range",
		},
		{
			label:       "invalid syntax float32",
			input:       "1.0 3.41+e38",
			decode:      floatDecode,
			errContains: "invalid syntax",
		},
	}

	testTestCases(t, cases)
}

func TestDecodeDate(t *testing.T) {
	timeDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got time.Time
		err := d.Decode(&got)
		return got, err
	}

	cases := []testcase{
		{
			label:  "RFC3339",
			input:  "2021-01-01T00:00:00Z",
			want:   func() interface{} { return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC) },
			decode: timeDecode,
		},
		{
			label:  "only year",
			input:  "2021",
			want:   func() interface{} { return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC) },
			decode: timeDecode,
		},
		{
			label:       "invalid string",
			input:       "not-a-date-string",
			decode:      timeDecode,
			errContains: "error decoding to time.Time",
		},
	}

	testTestCases(t, cases)
}

func TestDecodeDuration(t *testing.T) {
	s5, err := time.ParseDuration("5s")
	if err != nil {
		t.Fatal(err)
	}

	durationDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got time.Duration
		err := d.Decode(&got)
		return got, err
	}
	cases := []testcase{
		{
			label:  "5s",
			input:  "5s",
			decode: durationDecode,
			want:   func() interface{} { return s5 },
		},
		{
			label:       "invalid duration",
			input:       "not-a-duration-string",
			decode:      durationDecode,
			errContains: "invalid duration",
		},
	}

	testTestCases(t, cases)
}

func TestDecodeStruct(t *testing.T) {
	type person struct {
		Name   string
		Age    int
		Date   time.Time
		Active bool
	}

	structDecode := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var p person
		err := d.Decode(&p)
		return p, err
	}
	cases := []testcase{
		{
			label:  "tokens == fields",
			input:  "John 42 2021-01-01T00:00:00Z true",
			want:   func() interface{} { return person{"John", 42, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), true} },
			decode: structDecode,
		},
		{
			label:  "tokens < fields",
			input:  "John 42 2021-01-01T00:00:00Z",
			want:   func() interface{} { return person{"John", 42, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), false} },
			decode: structDecode,
		},
		{
			label:       "tokens > fields",
			input:       "John 42 2021-01-01T00:00:00Z true 87",
			decode:      structDecode,
			errContains: "too many tokens for struct strum_test.person",
		},
		{
			label:  "zero valued",
			input:  "John 42 2021-01-01T00:00:00Z",
			want:   func() interface{} { return person{"John", 42, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), false} },
			decode: structDecode,
		},
	}

	testTestCases(t, cases)
}

func TestDecodeTextUnmarshaler(t *testing.T) {
	bigratDecoder := func(t *testing.T, d *strum.Decoder) (interface{}, error) {
		var got *big.Rat
		err := d.Decode(&got)
		return got, err
	}
	cases := []testcase{
		{
			label:     "big.Rat",
			input:     "1/3",
			want:      func() interface{} { return big.NewRat(1, 3) },
			decode:    bigratDecoder,
			normalize: func(v interface{}) interface{} { return fmt.Sprintf("%v", v) },
		},
		{
			label:       "a/b",
			input:       "a/b",
			decode:      bigratDecoder,
			errContains: "cannot unmarshal",
		},
	}

	testTestCases(t, cases)
}

func TestUnsupportedType(t *testing.T) {
	r := bytes.NewBufferString("123")
	d := strum.NewDecoder(r)

	// Decode to struct has a deep type check
	type unsupported struct {
		C complex128
	}
	var u unsupported
	err := d.Decode(&u)
	errContains(t, err, "unsupported type complex128", "unsupported")
}

func TestPointers(t *testing.T) {
	type sxWithBoolHandle struct {
		B **bool
	}
	cases := []testcase{
		{
			label: "bool",
			input: "true",
			want:  func() interface{} { return true },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var b bool
				err := d.Decode(&b)
				return b, err
			},
		},
		{
			label: "*bool",
			input: "true",
			want:  func() interface{} { b := true; return &b },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var b bool
				pb := &b
				err := d.Decode(&pb)
				return pb, err
			},
		},
		{
			label: "*bool uninitialized",
			input: "true",
			want:  func() interface{} { b := true; return &b },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var pb *bool
				err := d.Decode(&pb)
				return pb, err
			},
		},
		{
			label: "**bool uninitialized",
			input: "true",
			want:  func() interface{} { b := true; pb := &b; return &pb },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var pb **bool
				err := d.Decode(&pb)
				return pb, err
			},
		},
		{
			label: "*struct with **bool uninitialized",
			input: "true",
			want:  func() interface{} { b := true; pb := &b; return &sxWithBoolHandle{B: &pb} },
			decode: func(t *testing.T, d *strum.Decoder) (interface{}, error) {
				var x *sxWithBoolHandle
				err := d.Decode(&x)
				return x, err
			},
		},
	}

	testTestCases(t, cases)
}
