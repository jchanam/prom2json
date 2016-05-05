// Copyright 2014 Prometheus Team
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

package prom2json

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/prometheus/common/expfmt"

	dto "github.com/prometheus/client_model/go"
)

// MetricFamily is a collection of metrics
type MetricFamily struct {
	Name    string        `json:"name"`
	Help    string        `json:"help"`
	Type    string        `json:"type"`
	Metrics []interface{} `json:"metrics,omitempty"` // Either metric or summary.
}

// Metric is single-value metric
type Metric struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  string            `json:"value"`
}

// Summary is a multiple-value metric
type Summary struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Quantiles map[string]string `json:"quantiles,omitempty"`
	Count     string            `json:"count"`
	Sum       string            `json:"sum"`
}

type Histogram struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Buckets map[string]string `json:"buckets,omitempty"`
	Count   string            `json:"count"`
	Sum     string            `json:"sum"`
}

func newMetricFamily(dtoMF *dto.MetricFamily) *MetricFamily {
	mf := &MetricFamily{
		Name:    dtoMF.GetName(),
		Help:    dtoMF.GetHelp(),
		Type:    dtoMF.GetType().String(),
		Metrics: make([]interface{}, len(dtoMF.Metric)),
	}
	for i, m := range dtoMF.Metric {
		if dtoMF.GetType() == dto.MetricType_SUMMARY {
			mf.Metrics[i] = Summary{
				Labels:    makeLabels(m),
				Quantiles: makeQuantiles(m),
				Count:     fmt.Sprint(m.GetSummary().GetSampleCount()),
				Sum:       fmt.Sprint(m.GetSummary().GetSampleSum()),
			}
		} else if dtoMF.GetType() == dto.MetricType_HISTOGRAM {
			mf.Metrics[i] = Histogram{
				Labels:  makeLabels(m),
				Buckets: makeBuckets(m),
				Count:   fmt.Sprint(m.GetHistogram().GetSampleCount()),
				Sum:     fmt.Sprint(m.GetSummary().GetSampleSum()),
			}
		} else {
			mf.Metrics[i] = Metric{
				Labels: makeLabels(m),
				Value:  fmt.Sprint(getValue(m)),
			}
		}
	}
	return mf
}

func getValue(m *dto.Metric) float64 {
	if m.Gauge != nil {
		return m.GetGauge().GetValue()
	}
	if m.Counter != nil {
		return m.GetCounter().GetValue()
	}
	if m.Untyped != nil {
		return m.GetUntyped().GetValue()
	}
	return 0.
}

func makeLabels(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, lp := range m.Label {
		result[lp.GetName()] = lp.GetValue()
	}
	return result
}

func makeQuantiles(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, q := range m.GetSummary().Quantile {
		result[fmt.Sprint(q.GetQuantile())] = fmt.Sprint(q.GetValue())
	}
	return result
}

func makeBuckets(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, b := range m.GetHistogram().Bucket {
		result[fmt.Sprint(b.GetUpperBound())] = fmt.Sprint(b.GetCumulativeCount())
	}
	return result
}

func fetchMetricFamilies(url string) ([]*dto.MetricFamily, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(url + " not HTTP 200 OK")
	}

	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}

	result := []*dto.MetricFamily{}
	for _, metric := range metrics {
		result = append(result, metric)
	}

	return result, nil
}

// Parse receives a prometheus metric url and return a parsed json string
func Parse(url string) (map[string][]string, error) {
	metrics, err := fetchMetricFamilies(url)

	if err != nil {
		return nil, err
	}

	response := []*MetricFamily{}
	for _, mf := range metrics {
		response = append(response, newMetricFamily(mf))
	}

	result := make(map[string][]string)

	for _, entry := range response {
		for _, metric := range entry.Metrics {
			if w, ok := metric.(Metric); ok {
				if labels, ok := w.Labels; ok {
					result[entry.Name] = append(labels)
				}
				result[entry.Name] = append(result[entry.Name], w.Value)
				continue
			}
			if w, ok := metric.(Summary); ok {
				result[entry.Name] = append(result[entry.Name], w.Sum)
				continue
			}
		}
	}

	if result == nil {
		return nil, errors.New("result is nil")
	}

	return result, nil
}
