# strum – String Unmarshaler

[![Go Reference](https://pkg.go.dev/badge/github.com/xdg-go/strum.svg)](https://pkg.go.dev/github.com/xdg-go/strum)
[![Go Report Card](https://goreportcard.com/badge/github.com/xdg-go/strum)](https://goreportcard.com/report/github.com/xdg-go/strum)
[![Github Actions](https://github.com/xdg-go/strum/actions/workflows/test.yml/badge.svg)](https://github.com/xdg-go/strum/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/xdg-go/strum/branch/master/graph/badge.svg)](https://codecov.io/gh/xdg-go/strum)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

The strum package provides line-oriented text decoding to simple Go structs.

* Splits on whitespace, a delimiter, a regular expression, or a custom
  tokenizer.
* Supports basic primitive types: strings, booleans, ints, uints
* Supports decoding RFC 3399 into `time.Time`

# Examples

```golang
func ExampleDecoder_Decode() {
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
			break
		}
		if err != nil {
			log.Fatalf("decoding error: %v", err)
		}
		fmt.Printf("%v\n", p)
	}

	// Output:
	// {John 42 true 2020-03-01 00:00:00 +0000 UTC}
	// {Jane 23 false 2022-02-22 00:00:00 +0000 UTC}
}
```

# Copyright and License

Copyright 2021 by David A. Golden. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License").
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
