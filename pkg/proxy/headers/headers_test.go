/*
 * Copyright 2018 The Trickster Authors
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

package headers

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/trickstercache/trickster/pkg/runtime"
	"github.com/trickstercache/trickster/pkg/timeseries"
)

func TestExtractHeader(t *testing.T) {

	headers := http.Header{}

	const appName = "trickster-test"
	const appVer = "tests"
	const appString = appName + " " + appVer

	runtime.ApplicationName = appName
	runtime.ApplicationVersion = appVer

	const testIP = "0.0.0.0"

	headers.Set(NameXForwardedFor, testIP)
	headers.Set(NameVia, appString)

	if h, ok := ExtractHeader(headers, NameXForwardedFor); !ok {
		t.Errorf("missing header %s", NameXForwardedFor)
	} else if h != testIP {
		t.Errorf(`expected "%s". got "%s"`, testIP, h)
	}

	if h, ok := ExtractHeader(headers, NameVia); !ok {
		t.Errorf("missing header %s", NameVia)
	} else if h != appString {
		t.Errorf(`expected "%s". got "%s"`, appString, h)
	}

	if _, ok := ExtractHeader(headers, NameAllowOrigin); ok {
		t.Errorf("unexpected header %s", NameAllowOrigin)
	}

}

func TestUpdateHeaders(t *testing.T) {
	headers := http.Header{"Foo1": {"foo"}, "Foo2": {"x"}, "Foo3": {"foo"}}
	expected := http.Header{"Foo1": {"bar"}, "Foo3": {"foo", "bar"}, "Foo4": {"bar"}, "Foo5": {"bar"}}

	UpdateHeaders(headers, nil)
	if len(headers) != 3 {
		t.Errorf("expected %d got %d", len(headers), 3)
	}

	UpdateHeaders(headers, map[string]string{"": "ineffectual", "foo1": "bar", "-foo2": "",
		"+foo3": "bar", "foo4": "bar", "+foo5": "bar", "-foo6": ""})
	if !reflect.DeepEqual(headers, expected) {
		fmt.Printf("mismatch\nexpected: %v\n     got: %v\n", expected, headers)
	}

}

func TestRemoveClientHeaders(t *testing.T) {

	headers := http.Header{}
	headers.Set(NameAcceptEncoding, "test")

	StripClientHeaders(headers)

	if _, ok := ExtractHeader(headers, NameAcceptEncoding); ok {
		t.Errorf("unexpected header %s", NameAcceptEncoding)
	}

}

func TestMerge(t *testing.T) {
	h1 := make(http.Header)
	h1.Set("test", "pass")
	h2 := make(http.Header)
	h2.Set("test2", "pass")

	Merge(h2, h1)
	if h2.Get("test") != "pass" {
		t.Errorf("expected 'pass' got '%s'", h2.Get("test"))
	}

	Merge(h2, nil)
	if h2.Get("test") != "pass" {
		t.Errorf("expected 'pass' got '%s'", h2.Get("test"))
	}

	h2["test2"] = make([]string, 0)

	Merge(h1, h2)
	if h1.Get("test") != "pass" {
		t.Errorf("expected 'pass' got '%s'", h1.Get("test"))
	}

}

func TestAddResponseHeaders(t *testing.T) {

	headers := http.Header{}
	runtime.ApplicationName = "trickster-test"
	runtime.ApplicationVersion = "tests"

	AddResponseHeaders(headers)

	if _, ok := headers[NameAllowOrigin]; !ok {
		t.Errorf("missing header %s", NameAllowOrigin)
	}

}

func TestSetResultsHeader(t *testing.T) {
	h := http.Header{}
	SetResultsHeader(h, "test-engine", "test-status", "test-ffstatus",
		timeseries.ExtentList{timeseries.Extent{Start: time.Unix(1, 0), End: time.Unix(2, 0)}})
	const expected = "engine=test-engine; status=test-status; fetched=[1:2]; ffstatus=test-ffstatus"
	if h.Get(NameTricksterResult) != expected {
		t.Errorf("expected %s got %s", expected, h.Get(NameTricksterResult))
	}
}

func TestSetResultsHeaderEmtpy(t *testing.T) {
	h := http.Header{}
	SetResultsHeader(h, "", "test-status", "test-ffstatus",
		timeseries.ExtentList{timeseries.Extent{Start: time.Unix(1, 0), End: time.Unix(2, 0)}})
	if len(h) > 0 {
		t.Errorf("Expected header length of %d", 0)
	}
}

func TestString(t *testing.T) {

	expected := "test: test\n\n"
	h := http.Header{"test": {"test"}}
	x := String(h)
	if x != expected {
		t.Errorf("expected %s got %s", expected, x)
	}

	expected = "\n\n"
	h = http.Header{}
	x = String(h)
	if x != expected {
		t.Errorf("expected %s got %s", expected, x)
	}

}

func TestLogString(t *testing.T) {

	expected := "{[test1:test],[test2:test2val]}"
	h := http.Header{"test1": {"test"}, "test2": {"test2val"}}
	x := LogString(h)
	if x != expected {
		t.Errorf("expected %s got %s", expected, x)
	}

	expected = "{}"
	h = http.Header{}
	x = LogString(h)
	if x != expected {
		t.Errorf("expected %s got %s", expected, x)
	}

	x = LogString(nil)
	if x != expected {
		t.Errorf("expected %s got %s", expected, x)
	}

}
