//go:generate ../../../tools/readme_config_includer/generator
package prometheus_client

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
	"regexp"	
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/internal"
	tlsint "github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/outputs"
	v1 "github.com/influxdata/telegraf/plugins/outputs/prometheus_client/v1"
	v2 "github.com/influxdata/telegraf/plugins/outputs/prometheus_client/v2"
)

var (
	InvalidNameCharRE  = regexp.MustCompile(`[^a-zA-Z0-9_:]`)
	ValidTagNameCharRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// DO NOT REMOVE THE NEXT TWO LINES! This is required to embed the sampleConfig data.
//go:embed sample.conf
var sampleConfig string

var (
	defaultListen             = ":9273"
	defaultPath               = "/metrics"
	defaultExpirationInterval = config.Duration(60 * time.Second)
)

type Collector interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
	Add(metrics []telegraf.Metric) error
}

type PrometheusClient struct {
	Listen             string          `toml:"listen"`
	MetricVersion      int             `toml:"metric_version"`
	BasicUsername      string          `toml:"basic_username"`
	BasicPassword      string          `toml:"basic_password"`
	IPRange            []string        `toml:"ip_range"`
	ExpirationInterval config.Duration `toml:"expiration_interval"`
	Path               string          `toml:"path"`
	CollectorsExclude  []string        `toml:"collectors_exclude"`
	StringAsLabel      bool            `toml:"string_as_label"`
	ExportTimestamp    bool            `toml:"export_timestamp"`
	tlsint.ServerConfig

	Log telegraf.Logger `toml:"-"`

	server    *http.Server
	url       *url.URL
	collector Collector
	wg        sync.WaitGroup
}

func (*PrometheusClient) SampleConfig() string {
	return sampleConfig
}

func (p *PrometheusClient) Init() error {
	defaultCollectors := map[string]bool{
		"gocollector": true,
		"process":     true,
	}
	for _, collector := range p.CollectorsExclude {
		delete(defaultCollectors, collector)
	}

	registry := prometheus.NewRegistry()
	for collector := range defaultCollectors {
		switch collector {
		case "gocollector":
			err := registry.Register(collectors.NewGoCollector())
			if err != nil {
				return err
			}
		case "process":
			err := registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized collector %s", collector)
		}
	}

	switch p.MetricVersion {
	default:
		fallthrough
	case 1:
		p.collector = v1.NewCollector(time.Duration(p.ExpirationInterval), p.StringAsLabel, p.Log)
		err := registry.Register(p.collector)
		if err != nil {
			return err
		}
	case 2:
		p.collector = v2.NewCollector(time.Duration(p.ExpirationInterval), p.StringAsLabel, p.ExportTimestamp)
		err := registry.Register(p.collector)
		if err != nil {
			return err
		}
	}

	ipRange := make([]*net.IPNet, 0, len(p.IPRange))
	for _, cidr := range p.IPRange {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("error parsing ip_range: %v", err)
		}

		ipRange = append(ipRange, ipNet)
	}

	authHandler := internal.AuthHandler(p.BasicUsername, p.BasicPassword, "prometheus", onAuthError)
	rangeHandler := internal.IPRangeHandler(ipRange, onError)
	promHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError})
	landingPageHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Telegraf Output Plugin: Prometheus Client "))
		if err != nil {
			p.Log.Errorf("Error occurred when writing HTTP reply: %v", err)
		}
	})

	mux := http.NewServeMux()
	if p.Path == "" {
		p.Path = "/metrics"
	}
	mux.Handle(p.Path, authHandler(rangeHandler(promHandler)))
	mux.Handle("/", authHandler(rangeHandler(landingPageHandler)))

	tlsConfig, err := p.TLSConfig()
	if err != nil {
		return err
	}

	p.server = &http.Server{
		Addr:      p.Listen,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	return nil
}

func (p *PrometheusClient) listen() (net.Listener, error) {
	if p.server.TLSConfig != nil {
		return tls.Listen("tcp", p.Listen, p.server.TLSConfig)
	}
	return net.Listen("tcp", p.Listen)
}

func (p *PrometheusClient) Connect() error {
	listener, err := p.listen()
	if err != nil {
		return err
	}

	scheme := "http"
	if p.server.TLSConfig != nil {
		scheme = "https"
	}

	p.url = &url.URL{
		Scheme: scheme,
		Host:   listener.Addr().String(),
		Path:   p.Path,
	}

	p.Log.Infof("Listening on %s", p.URL())

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := p.server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			p.Log.Errorf("Server error: %v", err)
		}
	}()

	return nil
}

func onAuthError(_ http.ResponseWriter) {
}

func onError(rw http.ResponseWriter, code int) {
	http.Error(rw, http.StatusText(code), code)
}

// URL returns the address the plugin is listening on.  If not listening
// an empty string is returned.
func (p *PrometheusClient) URL() string {
	if p.url != nil {
		return p.url.String()
	}
	return ""
}

func (p *PrometheusClient) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.server.Shutdown(ctx)
	p.wg.Wait()
	p.url = nil
	prometheus.Unregister(p.collector)
	return err
}

// func (p *PrometheusClient) Write(metrics []telegraf.Metric) error {
// 	return p.collector.Add(metrics)
// }
func (p *PrometheusClient) Write(metrics []telegraf.Metric) error {
	p.Lock()
	defer p.Unlock()

	now := p.now()

	for _, point := range Sorted(metrics) {
		tags := point.Tags()
		sampleID := CreateSampleID(tags)

		labels := make(map[string]string)
		for k, v := range tags {
			tName := Sanitize(k)
			if !IsValidTagName(tName) {
				continue
			}
			labels[tName] = v
		}

		// Prometheus doesn't have a string value type, so convert string
		// fields to labels if enabled.
		if p.StringAsLabel {
			for fn, fv := range point.Fields() {
				switch fv := fv.(type) {
				case string:
					tName := Sanitize(fn)
					if !IsValidTagName(tName) {
						continue
					}
					labels[tName] = fv
				}
			}
		}

		switch point.Type() {
		case telegraf.Summary:
			var mname string
			var sum float64
			var count uint64
			summaryvalue := make(map[float64]float64)
			for fn, fv := range point.Fields() {
				var value float64
				switch fv := fv.(type) {
				case int64:
					value = float64(fv)
				case uint64:
					value = float64(fv)
				case float64:
					value = fv
				default:
					continue
				}

				switch fn {
				case "sum":
					sum = value
				case "count":
					count = uint64(value)
				default:
					limit, err := strconv.ParseFloat(fn, 64)
					if err == nil {
						summaryvalue[limit] = value
					}
				}
			}
			sample := &Sample{
				Labels:       labels,
				SummaryValue: summaryvalue,
				Count:        count,
				Sum:          sum,
				Timestamp:    point.Time(),
				Expiration:   now.Add(p.ExpirationInterval.Duration),
			}
			mname = Sanitize(point.Name())

			if !IsValidTagName(mname) {
				continue
			}

			p.addMetricFamily(point, sample, mname, sampleID)

		case telegraf.Histogram:
			var mname string
			var sum float64
			var count uint64
			histogramvalue := make(map[float64]uint64)
			for fn, fv := range point.Fields() {
				var value float64
				switch fv := fv.(type) {
				case int64:
					value = float64(fv)
				case uint64:
					value = float64(fv)
				case float64:
					value = fv
				default:
					continue
				}

				switch fn {
				case "sum":
					sum = value
				case "count":
					count = uint64(value)
				default:
					limit, err := strconv.ParseFloat(fn, 64)
					if err == nil {
						histogramvalue[limit] = uint64(value)
					}
				}
			}
			sample := &Sample{
				Labels:         labels,
				HistogramValue: histogramvalue,
				Count:          count,
				Sum:            sum,
				Timestamp:      point.Time(),
				Expiration:     now.Add(p.ExpirationInterval.Duration),
			}
			mname = Sanitize(point.Name())

			if !IsValidTagName(mname) {
				continue
			}

			p.addMetricFamily(point, sample, mname, sampleID)

		default:
			for fn, fv := range point.Fields() {
				// Ignore string and bool fields.
				var value float64
				switch fv := fv.(type) {
				case int64:
					value = float64(fv)
				case uint64:
					value = float64(fv)
				case float64:
					value = fv
				default:
					continue
				}

				sample := &Sample{
					Labels:     labels,
					Value:      value,
					Timestamp:  point.Time(),
					Expiration: now.Add(p.ExpirationInterval.Duration),
				}

				// Special handling of value field; supports passthrough from
				// the prometheus input.
				var mname string
				switch point.Type() {
				case telegraf.Counter:
					if fn == "counter" {
						mname = Sanitize(point.Name())
					}
				case telegraf.Gauge:
					if fn == "gauge" {
						mname = Sanitize(point.Name())
					}
				}
				if mname == "" {
					if fn == "value" {
						mname = Sanitize(point.Name())
					} else {
						mname = Sanitize(fmt.Sprintf("%s_%s", point.Name(), fn))
					}
				}
				if !IsValidTagName(mname) {
					continue
				}
				p.addMetricFamily(point, sample, mname, sampleID)

			}
		}
	}
	return nil
}

func init() {
	outputs.Add("prometheus_client", func() telegraf.Output {
		return &PrometheusClient{
			Listen:             defaultListen,
			Path:               defaultPath,
			ExpirationInterval: defaultExpirationInterval,
			StringAsLabel:      true,
		}
	})
}

func Sanitize(value string) string {
	return InvalidNameCharRE.ReplaceAllString(value, "_")
}

func IsValidTagName(tag string) bool {
	return ValidTagNameCharRE.MatchString(tag)
}

// Sorted returns a copy of the metrics in time ascending order.  A copy is
// made to avoid modifying the input metric slice since doing so is not
// allowed.
func Sorted(metrics []telegraf.Metric) []telegraf.Metric {
	batch := make([]telegraf.Metric, 0, len(metrics))
	for i := len(metrics) - 1; i >= 0; i-- {
		batch = append(batch, metrics[i])
	}
	sort.Slice(batch, func(i, j int) bool {
		return batch[i].Time().Before(batch[j].Time())
	})
	return batch
}
