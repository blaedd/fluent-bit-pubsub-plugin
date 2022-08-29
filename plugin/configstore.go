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
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/rs/zerolog"
)

var (
	// sn is a Regexp for potentially sensitive fields we shouldn't log
	sn = regexp.MustCompile(`(?i:pass|secret|key|hash)`)
	// configKeySet is a variable pointing to the function we use to retrieve config keys from the actual
	// configuration file. Mainly here so we can easily mock it out for tests.
	configKeyGet = output.FLBPluginConfigKey
)

// FLBConfigStore provides access to the fluent-bit configuration for the plugin.
type FLBConfigStore struct {
	l   *zerolog.Logger
	ctx unsafe.Pointer
}

func NewFLBConfigStore(ctx unsafe.Pointer, l *zerolog.Logger) FLBConfigStore {
	return FLBConfigStore{
		l:   l,
		ctx: ctx,
	}
}

func (f *FLBConfigStore) getKey(name string) string {
	return configKeyGet(f.ctx, name)
}

func (f *FLBConfigStore) logGetKey(n string, rv interface{}, ok bool) {
	if e := f.l.Debug(); !e.Enabled() {
		return
	}
	var (
		evt = zerolog.Dict().Str("name", n).Bool("found", ok)
		msg string
	)

	if ok {
		if sn.Match([]byte(n)) {
			evt = evt.Str("value", "********")
		} else {
			switch v := rv.(type) {
			case bool:
				evt = evt.Bool("value", v)
			case int:
				evt = evt.Int("value", v)
			case string:
				evt = evt.Str("value", v)
			case []string:
				evt = evt.Strs("value", v)
			case time.Duration:
				evt = evt.Dur("value", v)
			default:
				evt = evt.Interface("value", rv)
			}
		}
		// Redact sensitive sounding fields.
		msg = "found config key"
	} else {
		msg = "did not find config key"
	}
	f.l.Debug().Dict("configkey", evt).Msg(msg)
}

// Bool retrieves a boolean from the plugin configuration.
//
// The value and if the value was found are returned.
func (f *FLBConfigStore) Bool(name string) (bool, bool) {
	sv := f.getKey(name)
	b, err := strconv.ParseBool(sv)
	ok := err == nil && sv != ""
	f.logGetKey(name, b, ok)
	return b, ok
}

// Duration retrieves a [time.Duration] from the plugin configuration.
//
// The value and if the value was found are returned.
func (f *FLBConfigStore) Duration(name string) (time.Duration, bool) {
	sv := f.getKey(name)
	d, err := time.ParseDuration(sv)
	ok := err == nil && sv != ""
	f.logGetKey(name, d, ok)
	return d, ok
}

// Int retrieves an integer from the plugin configuration.
//
// The value and if the value was found are returned.
func (f *FLBConfigStore) Int(name string) (int, bool) {
	sv := f.getKey(name)
	i, err := strconv.Atoi(sv)
	ok := err == nil && sv != ""
	f.logGetKey(name, i, ok)
	return i, ok
}

// String retrieves a string from the plugin configuration.
//
// The value and if the value was found are returned.
func (f *FLBConfigStore) String(name string) (string, bool) {
	sv := f.getKey(name)
	ok := sv != ""
	f.logGetKey(name, sv, ok)
	return sv, ok
}

// Strings retrieves a list of strings from the plugin configuration.
//
// These should be comma seperated in the configuration file. The value and if the value was found are returned.
func (f *FLBConfigStore) Strings(name string) ([]string, bool) {
	sv := f.getKey(name)
	spt := func(c rune) bool {
		return unicode.IsSpace(c) || c == ','
	}
	ss := strings.FieldsFunc(sv, spt)
	ok := len(ss) > 0
	f.logGetKey(name, ss, ok)
	return ss, ok
}
