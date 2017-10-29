// Copyright 2016 Google Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package embedmd

import (
	"testing"
)

const content = `
package main

import "fmt"

func main() {
		// START test
		fmt.Println("hello, test")
		// END test

		// START a
		fmt.Println()
		// END a
}
`

func TestExtract(t *testing.T) {
	tc := []struct {
		name   string
		sample string
		out    string
	}{
		{
			name:   "start and end comment",
			sample: "test",
			out:    "START test\n\t\tfmt.Println(\"hello, test\")\n\t\t// END test",
		},
		{
			name:   "a",
			sample: "a",
			out:    "START a\n\t\tfmt.Println()\n\t\t// END a",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			b, err := extract([]byte(content), tt.sample)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.out {
				t.Errorf("case [%s]: expected extracting %q; got %q", tt.name, tt.out, b)
			}
		})
	}
}
