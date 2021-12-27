# strum – String Unmarshaler

[![Go Reference](https://pkg.go.dev/badge/github.com/xdg-go/strum.svg)](https://pkg.go.dev/github.com/xdg-go/strum)
[![Go Report Card](https://goreportcard.com/badge/github.com/xdg-go/strum)](https://goreportcard.com/report/github.com/xdg-go/strum)
[![Github Actions](https://github.com/xdg-go/strum/actions/workflows/test.yml/badge.svg)](https://github.com/xdg-go/strum/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/xdg-go/strum/branch/master/graph/badge.svg)](https://codecov.io/gh/xdg-go/strum)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

The strum package provides line-oriented text decoding into simple Go
variables, slices, and structs.

* Splits on whitespace, a delimiter, a regular expression, or a custom
  tokenizer.
* Supports basic primitive types: strings, booleans, ints, uints, floats.
* Supports decoding RFC 3399 into `time.Time`.
* Supports decoding time.Duration.
* Supports `encoding.TextUnmarshaler` types.
* Decodes a line into a single variable, a slice, or a struct.
* Decodes all lines into a slice of the above.

# Synopsis

```golang
	d := strum.NewDecoder(os.Stdin)

	// Decode a line to a single int
	var x int
	err = d.Decode(&x)

	// Decode a line to a slice of int
	var xs []int
	err = d.Decode(&xs)

	// Decode a line to a struct
	type person struct {
		Name string
		Age  int
	}
	var p person
	err = d.Decode(&p)

	// Decode all lines to a slice of struct
	var people []person
	err = d.DecodeAll(&people)
```

# Copyright and License

Copyright 2021 by David A. Golden. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License").
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
