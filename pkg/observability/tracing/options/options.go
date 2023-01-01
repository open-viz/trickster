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

package options

import (
	jaegeropts "github.com/trickstercache/trickster/v2/pkg/observability/tracing/exporters/jaeger/options"
	stdoutopts "github.com/trickstercache/trickster/v2/pkg/observability/tracing/exporters/stdout/options"
	"github.com/trickstercache/trickster/v2/pkg/util/copiers"
	"github.com/trickstercache/trickster/v2/pkg/util/yamlx"
)

// Options is a Tracing Options collection
type Options struct {
	Name          string            `json:"-"`
	Provider      string            `json:"provider,omitempty"`
	ServiceName   string            `json:"service_name,omitempty"`
	CollectorURL  string            `json:"collector_url,omitempty"`
	CollectorUser string            `json:"collector_user,omitempty"`
	CollectorPass string            `json:"collector_pass,omitempty"`
	SampleRate    float64           `json:"sample_rate,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	OmitTagsList  []string          `json:"omit_tags,omitempty"`

	StdOutOptions *stdoutopts.Options `json:"stdout,omitempty"`
	JaegerOptions *jaegeropts.Options `json:"jaeger,omitempty"`

	OmitTags map[string]interface{} `json:"-"`
	// for tracers that don't support WithProcess (e.g., Zipkin)
	attachTagsToSpan bool `json:"-"`
}

// New returns a new *Options with the default values
func New() *Options {
	return &Options{
		Provider:      DefaultTracerProvider,
		ServiceName:   DefaultTracerServiceName,
		StdOutOptions: &stdoutopts.Options{},
		JaegerOptions: &jaegeropts.Options{},
	}
}

// Clone returns an exact copy of a tracing config
func (o *Options) Clone() *Options {
	var so *stdoutopts.Options
	if o.StdOutOptions != nil {
		so = o.StdOutOptions.Clone()
	}
	var jo *jaegeropts.Options
	if o.JaegerOptions != nil {
		jo = o.JaegerOptions.Clone()
	}
	return &Options{
		Name:             o.Name,
		Provider:         o.Provider,
		ServiceName:      o.ServiceName,
		CollectorURL:     o.CollectorURL,
		CollectorUser:    o.CollectorUser,
		CollectorPass:    o.CollectorPass,
		SampleRate:       o.SampleRate,
		Tags:             copiers.CopyStringLookup(o.Tags),
		OmitTags:         copiers.CopyLookup(o.OmitTags),
		OmitTagsList:     copiers.CopyStrings(o.OmitTagsList),
		StdOutOptions:    so,
		JaegerOptions:    jo,
		attachTagsToSpan: o.attachTagsToSpan,
	}
}

// ProcessTracingOptions enriches the configuration data of the provided Tracing Options collection
func ProcessTracingOptions(mo map[string]*Options, metadata yamlx.KeyLookup) {
	if len(mo) == 0 {
		return
	}
	for k, v := range mo {
		if metadata != nil {
			if !metadata.IsDefined("tracing", k, "sample_rate") {
				v.SampleRate = 1
			}
			if !metadata.IsDefined("tracing", k, "service_name") {
				v.ServiceName = DefaultTracerServiceName
			}
			if !metadata.IsDefined("tracing", k, "provider") {
				v.Provider = DefaultTracerProvider
			}
		}
		v.generateOmitTags()
		v.setAttachTags()
	}
}

func (o *Options) generateOmitTags() {
	o.OmitTags = copiers.LookupFromStrings(o.OmitTagsList)
}

// AttachTagsToSpan indicates that Tags should be attached to the span
func (o *Options) AttachTagsToSpan() bool {
	return o.attachTagsToSpan
}

func (o *Options) setAttachTags() {
	if o.Provider == "zipkin" && o.Tags != nil && len(o.Tags) > 0 {
		o.attachTagsToSpan = true
	}
}
