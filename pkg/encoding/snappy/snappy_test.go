/*
 * Copyright 2018 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package snappy

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestDecodeEncode(t *testing.T) {
	const expected = "trickster"
	b, err := Encode([]byte(expected))
	if err != nil {
		t.Error(err)
	}
	b, err = Decode(b)
	if err != nil {
		t.Error(err)
	}
	if string(b) != expected {
		t.Errorf("expected %s got %s", expected, string(b))
	}
}

func TestNewDecoder(t *testing.T) {
	const expected = "trickster"
	b, err := Encode([]byte(expected))
	if err != nil {
		t.Error(err)
	}
	r := bytes.NewReader(b)
	dec := NewDecoder(r)
	if dec == nil {
		t.Error("expected non-nil decoder")
	}
}

func TestNewEncoder(t *testing.T) {
	w := httptest.NewRecorder()
	enc := NewEncoder(w, 0)
	if enc == nil {
		t.Error("expected non-nil encoder")
	}

	w = httptest.NewRecorder()
	enc = NewEncoder(w, 1)
	if enc == nil {
		t.Error("expected non-nil encoder")
	}

	w = httptest.NewRecorder()
	enc = NewEncoder(w, 4)
	if enc == nil {
		t.Error("expected non-nil encoder")
	}

	w = httptest.NewRecorder()
	enc = NewEncoder(w, 9)
	if enc == nil {
		t.Error("expected non-nil encoder")
	}
}
