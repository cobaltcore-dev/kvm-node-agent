/*
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	histogramMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "node_reconcile_duration",
			Help:    "Duration of node reconcile.",
			Buckets: []float64{0.1, .25, .5, 0.75, 1, 2.5, 5, 10},
		},
		[]string{"node"},
	)
	counterMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "node_reconcile_count",
			Help: "How many reconcile requests processed.",
		},
		[]string{"node", "error"},
	)
)

func init() {
	metrics.Registry.MustRegister(histogramMetric)
	metrics.Registry.MustRegister(counterMetric)
}
