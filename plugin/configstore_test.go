/*
 * Copyright 2022  David MacKinnon (blaedd@gmail.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"). You may
 * not use this file except in compliance with the License. A copy of the
 * License is located at
 *
 * https://www.apache.org/licenses/LICENSE-2.0.txt
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package plugin

import (
	"testing"
	"time"
	"unsafe"

	"github.com/rs/zerolog"
)

func TestFLBConfigStore_Bool(t *testing.T) {
	l := zerolog.Nop()

	type testData struct {
		cv   string
		want bool
		ok   bool
	}

	testMap := map[string]testData{
		"strOk":   {"true", true, true},
		"strNok":  {"false", false, true},
		"strErr":  {"hello", false, false},
		"intOk":   {"1", true, true},
		"intNok":  {"0", false, true},
		"intNeg":  {"-1", false, false},
		"intGt":   {"2", false, false},
		"noValue": {"", false, false},
	}

	origGetKey := configKeyGet
	configKeyGet = func(ctx unsafe.Pointer, name string) string {
		if val, ok := testMap[name]; ok {
			return val.cv
		}
		return ""
	}

	for k, tt := range testMap {
		t.Run(k, func(t *testing.T) {
			f := &FLBConfigStore{
				ctx: nil,
				l:   &l,
			}
			got, got1 := f.Bool(k)
			if got != tt.want {
				t.Errorf("Bool() val = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("Bool() ok = %v, want_ok %v", got1, tt.ok)
			}
		})
	}
	configKeyGet = origGetKey
}

func TestFLBConfigStore_Duration(t *testing.T) {
	l := zerolog.Nop()

	type testData struct {
		cv   string
		want time.Duration
		ok   bool
	}
	testMap := make(map[string]testData)

	ti := map[string]struct {
		v  string
		ok bool
	}{
		"seconds": {"1s", true},
		"hours":   {"3h1s", true},
		// parseDuration doesn't support days
		"days":      {"1d3h", false},
		"invalid":   {"1", false},
		"noValue":   {"", false},
		"setToZero": {"0s", true},
	}
	for k, v := range ti {
		d, _ := time.ParseDuration(v.v)
		testMap[k] = testData{v.v, d, v.ok}
	}
	for _, v := range testMap {
		d, err := time.ParseDuration(v.cv)
		if err != nil {
			v.ok = false
		} else {
			v.ok = true
		}
		v.want = d

	}
	origGetKey := configKeyGet
	configKeyGet = func(ctx unsafe.Pointer, name string) string {
		if val, ok := testMap[name]; ok {
			return val.cv
		}
		return ""
	}

	for k, tt := range testMap {
		t.Run(k, func(t *testing.T) {
			f := &FLBConfigStore{
				ctx: nil,
				l:   &l,
			}
			got, got1 := f.Duration(k)
			if got != tt.want {
				t.Errorf("Duration() val = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("Duration() ok = %v, want_ok %v", got1, tt.ok)
			}
		})
	}
	configKeyGet = origGetKey
}

func TestFLBConfigStore_Int(t *testing.T) {
	l := zerolog.Nop()

	type testData struct {
		cv   string
		want int
		ok   bool
	}

	testMap := map[string]testData{
		"positive integer": {"1", 1, true},
		"negative integer": {"-1", -1, true},
		"zero":             {"0", 0, true},
		"unset":            {"", 0, false},
		"reallybigint":     {"30724897987981", 30724897987981, true},
		"float":            {"23.5", 0, false},
		"exp notation":     {"1e6", 0, false},
	}

	origGetKey := configKeyGet
	configKeyGet = func(ctx unsafe.Pointer, name string) string {
		if val, ok := testMap[name]; ok {
			return val.cv
		}
		return ""
	}

	for k, tt := range testMap {
		t.Run(k, func(t *testing.T) {
			f := &FLBConfigStore{
				ctx: nil,
				l:   &l,
			}
			got, got1 := f.Int(k)
			if got != tt.want {
				t.Errorf("Int() val = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("Int() ok = %v, want_ok %v", got1, tt.ok)
			}
		})
	}
	configKeyGet = origGetKey
}

func TestFLBConfigStore_String(t *testing.T) {
	l := zerolog.Nop()

	type testData struct {
		cv   string
		want string
		ok   bool
	}

	testMap := map[string]testData{
		"astring": {"astring", "astring", true},
		"notSet":  {"", "", false},
	}

	origGetKey := configKeyGet
	configKeyGet = func(ctx unsafe.Pointer, name string) string {
		if val, ok := testMap[name]; ok {
			return val.cv
		}
		return ""
	}

	for k, tt := range testMap {
		t.Run(k, func(t *testing.T) {
			f := &FLBConfigStore{
				ctx: nil,
				l:   &l,
			}
			got, got1 := f.String(k)
			if got != tt.want {
				t.Errorf("String() val = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("String() ok = %v, want_ok %v", got1, tt.ok)
			}
		})
	}
	configKeyGet = origGetKey
}

func TestFLBConfigStore_Strings(t *testing.T) {
	l := zerolog.Nop()

	type testData struct {
		cv   string
		want []string
		ok   bool
	}

	testMap := map[string]testData{
		"aString":               {"astring", []string{"astring"}, true},
		"notSet":                {"", nil, false},
		"manyStrings":           {"val1,val2,val3,val4", []string{"val1", "val2", "val3", "val4"}, true},
		"manyStringsWithSpaces": {" val1,  val2 ,val3 , val4", []string{"val1", "val2", "val3", "val4"}, true},
	}

	origGetKey := configKeyGet
	configKeyGet = func(ctx unsafe.Pointer, name string) string {
		if val, ok := testMap[name]; ok {
			return val.cv
		}
		return ""
	}

	for k, tt := range testMap {
		t.Run(k, func(t *testing.T) {
			f := &FLBConfigStore{
				ctx: nil,
				l:   &l,
			}
			got, got1 := f.Strings(k)
			gotLen := len(got)
			wantLen := len(tt.want)
			if gotLen != wantLen {
				t.Errorf("Strings() val = %v, want %v", got, tt.want)
			} else {
				gotMap := make(map[string]uint8)
				for _, v := range got {
					gotMap[v] = 0
				}
				for _, w := range tt.want {
					if _, ok := gotMap[w]; !ok {
						t.Errorf("Strings() val = %v, want %v", got, tt.want)
						break
					}
				}
			}
			if got1 != tt.ok {
				t.Errorf("Strings() ok = %v, want_ok %v", got1, tt.ok)
			}
		})
	}
	configKeyGet = origGetKey
}
