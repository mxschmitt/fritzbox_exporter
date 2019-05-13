package main

// Copyright 2016 Nils Decker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mxschmitt/fritzbox_exporter/pkg/fritzboxmetrics"
	"github.com/prometheus/client_golang/prometheus"
)

const serviceLoadRetryTime = 1 * time.Minute

var (
	collectErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "fritzbox_exporter_collect_errors",
		Help: "Number of collection errors.",
	})
)

type FritzboxCollector struct {
	Gateway  string
	Port     uint16
	Username string
	Password string

	sync.Mutex // protects Root
	Root       *fritzboxmetrics.Root
}

// LoadServices tries to load the service information. Retries until success.
func (fc *FritzboxCollector) LoadServices() {
	for {
		root, err := fritzboxmetrics.LoadServices(fc.Gateway, fc.Port, fc.Username, fc.Password)
		if err != nil {
			fmt.Printf("cannot load services: %v\n", err)
			// Sleep so long how often the metrics should be fetched
			time.Sleep(serviceLoadRetryTime)
			continue
		}

		fmt.Println("services loaded")

		fc.Lock()
		fc.Root = root
		fc.Unlock()
		return
	}
}

func (fc *FritzboxCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range metrics {
		ch <- m.Desc
	}
}

func (fc *FritzboxCollector) Collect(ch chan<- prometheus.Metric) {
	fc.Lock()
	root := fc.Root
	fc.Unlock()

	if root == nil {
		// Services not loaded yet
		return
	}

	var lastService string
	var lastMethod string
	var lastResult fritzboxmetrics.Result

	for _, m := range metrics {
		if m.Service != lastService || m.Action != lastMethod {
			service, ok := root.Services[m.Service]
			if !ok {
				// TODO
				fmt.Println("cannot find service", m.Service)
				fmt.Println(root.Services)
				continue
			}
			action, ok := service.Actions[m.Action]
			if !ok {
				// TODO
				fmt.Println("cannot find action", m.Action)
				continue
			}

			var err error
			lastResult, err = action.Call()
			if err != nil {
				log.Printf("could not call action %s: %v", action.Name, err)
				collectErrors.Inc()
				continue
			}
		}

		val, ok := lastResult[m.Result]
		if !ok {
			fmt.Println("result not found", m.Result)
			collectErrors.Inc()
			continue
		}

		var floatval float64
		switch tval := val.(type) {
		case uint64:
			floatval = float64(tval)
		case bool:
			if tval {
				floatval = 1
			} else {
				floatval = 0
			}
		case string:
			if tval == m.OkValue {
				floatval = 1
			} else {
				floatval = 0
			}
		default:
			fmt.Println("unknown", val)
			collectErrors.Inc()
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			m.Desc,
			m.MetricType,
			floatval,
		)
	}
}
