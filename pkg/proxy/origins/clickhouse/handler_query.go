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

package clickhouse

import (
	"net/http"
	"strings"

	"github.com/trickstercache/trickster/pkg/proxy/engines"
	"github.com/trickstercache/trickster/pkg/proxy/params"
	"github.com/trickstercache/trickster/pkg/proxy/urls"
)

// QueryHandler handles timeseries requests for ClickHouse and processes them through the delta proxy cache
func (c *Client) QueryHandler(w http.ResponseWriter, r *http.Request) {

	qp, _, isBody := params.GetRequestValues(r)
	q := strings.ToLower(qp.Get(upQuery))
	// if it's not a select statement, just proxy it instead
	if isBody || (!strings.Contains(q, "select ") &&
		(!strings.HasSuffix(q, " format json"))) {
		c.ProxyHandler(w, r)
		return
	}

	r.URL = urls.BuildUpstreamURL(r, c.baseUpstreamURL)
	engines.DeltaProxyCacheRequest(w, r)
}
