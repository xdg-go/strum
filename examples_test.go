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
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/xdg-go/strum"
)

func ExampleDecoder_Decode_struct() {
	type person struct {
		Name   string
		Age    int
		Active bool
		Joined time.Time
	}

	lines := []string{
		"John 42 true  2020-03-01T00:00:00Z",
		"Jane 23 false 2022-02-22T00:00:00Z",
	}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	for {
		var p person
		err := d.Decode(&p)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(p)
	}

	// Output:
	// {John 42 true 2020-03-01 00:00:00 +0000 UTC}
	// {Jane 23 false 2022-02-22 00:00:00 +0000 UTC}
}

func ExampleDecoder_DecodeAll_struct() {
	type person struct {
		Name   string
		Age    int
		Active bool
		Joined time.Time
	}

	lines := []string{
		"John 42 true  2020-03-01T00:00:00Z",
		"Jane 23 false 2022-02-22T00:00:00Z",
	}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var people []person
	err := d.DecodeAll(&people)
	if err != nil {
		log.Fatalf("decoding error: %v", err)
	}

	for _, p := range people {
		fmt.Printf("%v\n", p)
	}

	// Output:
	// {John 42 true 2020-03-01 00:00:00 +0000 UTC}
	// {Jane 23 false 2022-02-22 00:00:00 +0000 UTC}
}

func ExampleDecoder_DecodeAll_ints() {
	lines := []string{
		"42",
		"23",
	}

	r := bytes.NewBufferString(strings.Join(lines, "\n"))
	d := strum.NewDecoder(r)

	var xs []int
	err := d.DecodeAll(&xs)
	if err != nil {
		log.Fatalf("decoding error: %v", err)
	}

	for _, x := range xs {
		fmt.Printf("%d\n", x)
	}

	// Output:
	// 42
	// 23
}

func ExampleDecoder_WithTokenRegexp() {
	type jeans struct {
		Color  string
		Waist  int
		Inseam int
	}

	text := "Blue 36x32"
	r := bytes.NewBufferString(text)

	re := regexp.MustCompile(`^(\S+)\s+(\d+)x(\d+)`)
	d := strum.NewDecoder(r).WithTokenRegexp(re)

	var j jeans
	err := d.Decode(&j)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}

	fmt.Println(j)

	// Output:
	// {Blue 36 32}
}

func ExampleDecoder_WithSplitOn() {
	type person struct {
		Last  string
		First string
	}

	text := "Doe,John"
	r := bytes.NewBufferString(text)

	d := strum.NewDecoder(r).WithSplitOn(",")

	var p person
	err := d.Decode(&p)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}

	fmt.Println(p)

	// Output:
	// {Doe John}
}

func Example_synopsis() {
	var err error
	d := strum.NewDecoder(os.Stdin)

	// Decode a line to a single int
	var x int
	err = d.Decode(&x)
	if err != nil {
		log.Fatal(err)
	}

	// Decode a line to a slice of int
	var xs []int
	err = d.Decode(&xs)
	if err != nil {
		log.Fatal(err)
	}

	// Decode a line to a struct
	type person struct {
		Name string
		Age  int
	}
	var p person
	err = d.Decode(&p)
	if err != nil {
		log.Fatal(err)
	}

	// Decode all lines to a slice of struct
	var people []person
	err = d.DecodeAll(&people)
	if err != nil {
		log.Fatal(err)
	}
}
