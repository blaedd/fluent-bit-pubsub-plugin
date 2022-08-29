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

// Most of the code in this file is adapted from
//https://github.com/fluent/fluent-bit-go/blob/0be1ffb0c49b503cb6dca256f6e3d3357d242e53/output/decoder.go

import "C"
import (
	"encoding/binary"
	"errors"
	"reflect"
	"time"
	"unsafe"

	"github.com/ugorji/go/codec"
)

// An FLBRecordReader decodes a MsgPack record from fluent-bit
type FLBRecordReader struct {
	handle *codec.MsgpackHandle
	mpdec  *codec.Decoder
}
type FLBTime struct {
	time.Time
}

// ReadExt handles decoding the MsgPack extension
func (t FLBTime) ReadExt(i interface{}, b []byte) {
	out := i.(*FLBTime)
	sec := binary.BigEndian.Uint32(b)
	usec := binary.BigEndian.Uint32(b[4:])
	out.Time = time.Unix(int64(sec), int64(usec))
}

func (t FLBTime) WriteExt(interface{}) []byte {
	panic("unsupported")
}

// NewFLBRecordReader creates a new FLBRecordReader, and initializes the MsgPack handler and decoder.
func NewFLBRecordReader() (*FLBRecordReader, error) {
	mh := new(codec.MsgpackHandle)
	err := mh.SetBytesExt(reflect.TypeOf(FLBTime{}), 0, &FLBTime{})
	if err != nil {
		return nil, err
	}
	mpdec := codec.NewDecoderBytes([]byte{}, mh)
	return &FLBRecordReader{handle: mh, mpdec: mpdec}, nil
}

// ResetReader resets the MsgPack decoder contained in the FLBRecordReader, readying it to decode another record.
func (r *FLBRecordReader) ResetReader(data unsafe.Pointer, length int) {
	b := C.GoBytes(data, C.int(length))
	r.mpdec.ResetBytes(b)
}

// taken from https://github.com/tanakarian/fluent-bit-google-pubsub-out/blob/4a6024923388bd1f0913f99058293775d98ff966/converter.go
func makeJSONMap(record map[interface{}]interface{}) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	for k, v := range record {
		switch t := v.(type) {
		case []byte:
			// avoid json.Marshall encodes map's value to base64strings.
			jsonMap[k.(string)] = string(t)
		// nested json
		case map[interface{}]interface{}:
			value := makeJSONMap(t)
			jsonMap[k.(string)] = value
		default:
			jsonMap[k.(string)] = t
		}
	}
	return jsonMap
}

// ReadRecord reads the next record from bytes provided by fluent-bit.
//
// These records are encoded as [ts, record] slices. ReadRecord converts these to time.Time and map[string]interface{}
// for ready encoding as JSON and/or PubSub attributes.
func (r *FLBRecordReader) ReadRecord() (time.Time, map[string]interface{}, error) {
	var m interface{}

	err := r.mpdec.Decode(&m)
	if err != nil {
		return time.Time{}, nil, err
	}
	slice := reflect.ValueOf(m)
	if slice.Kind() != reflect.Slice || slice.Len() != 2 {
		return time.Time{}, nil, errors.New("unexpected or malformed data")
	}
	ts := slice.Index(0).Interface().(FLBTime)
	mapData := makeJSONMap(slice.Index(1).Interface().(map[interface{}]interface{}))

	return ts.Time, mapData, nil
}
