// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package strum_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xdg-go/strum"
)

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
