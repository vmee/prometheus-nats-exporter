// Copyright 2017-2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package collector has various collector utilities and implementations.
package collector

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

func isHealthzEndpoint(system, endpoint string) bool {
	return system == CoreSystem && endpoint == "healthz"
}

type healthzCollector struct {
	sync.Mutex

	httpClient *http.Client
	servers    []*CollectedServer

	state *prometheus.Desc
}

func newHealthzCollector(system, endpoint string, servers []*CollectedServer) prometheus.Collector {
	nc := &healthzCollector{
		httpClient: http.DefaultClient,
		state: prometheus.NewDesc(
			prometheus.BuildFQName(system, endpoint, "state"),
			"state",
			[]string{"server_id"},
			nil,
		),
	}

	nc.servers = make([]*CollectedServer, len(servers))
	for i, s := range servers {
		nc.servers[i] = &CollectedServer{
			ID:  s.ID,
			URL: s.URL + "/streaming/serverz",
		}
	}

	return nc
}

func (nc *healthzCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.state
}

// Collect gathers the server health state metrics.
func (nc *healthzCollector) Collect(ch chan<- prometheus.Metric) {
	for _, server := range nc.servers {
		var resp StreamingServerz
		var state float64 = 1
		if err := getMetricURL(nc.httpClient, server.URL, &resp); err != nil {
			Debugf("ignoring server %s: %v", server.ID, err)
			state = 0
		}

		if resp.ServerID == "" {
			state = 0
		}

		ch <- prometheus.MustNewConstMetric(nc.state, prometheus.GaugeValue, float64(state), server.ID)
	}
}
