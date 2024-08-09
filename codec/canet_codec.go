// Copyright 2015-2019 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"fmt"
	"reflect"

	"github.com/andeya/goutil"
)

// canet data codec name and id
const (
	NAME_CANET = "canet"
	ID_CANET   = 'c'
)

func init() {
	Reg(new(CanetCodec))
}

// CanetCodec canet data codec
type CanetCodec struct{}

// Name returns codec name.
func (CanetCodec) Name() string {
	return NAME_CANET
}

// ID returns codec id.
func (CanetCodec) ID() byte {
	return ID_CANET
}

// Marshal returns the string encoding of v.
func (CanetCodec) Marshal(v interface{}) ([]byte, error) {
	var b []byte
	switch vv := v.(type) {
	case nil:
	case string:
		b = goutil.StringToBytes(vv)
	case *string:
		b = goutil.StringToBytes(*vv)
	case []byte:
		b = vv
	case *[]byte:
		b = *vv
	default:
		s, ok := formatProperType(reflect.ValueOf(v))
		if !ok {
			return nil, fmt.Errorf("canet codec: %T can not be directly converted to []byte type", v)
		}
		b = goutil.StringToBytes(s)
	}
	return b, nil
}

// Unmarshal parses the string-encoded data and stores the result
// in the value pointed to by v.
func (CanetCodec) Unmarshal(data []byte, v interface{}) error {
	switch s := v.(type) {
	case nil:
		return nil
	case *string:
		*s = string(data)
	case []byte:
		copy(s, data)
	case *[]byte:
		if length := len(data); cap(*s) < length {
			*s = make([]byte, length)
		} else {
			*s = (*s)[:length]
		}
		copy(*s, data)
	default:
		if !parseProperType(data, reflect.ValueOf(v)) {
			return fmt.Errorf("canet codec: []byte can not be directly converted to %T type", v)
		}
	}
	return nil
}
