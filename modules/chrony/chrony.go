package chrony

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/netdata/go.d.plugin/agent/module"
	"net"
	"time"
)

type (
	Config struct {
		Protocol     string               `yaml:"protocol"`
		Address      string               `yaml:"address"`
		Timeout      int                  `yaml:"timeout"` // Millisecond
		SentryConfig sentry.ClientOptions `yaml:",inline"`
	}

	// Chrony is the main collector for chrony
	Chrony struct {
		module.Base   // should be embedded by every module
		Config        `yaml:",inline"`
		chronyVersion uint8
		latestSource  net.IP
		conn          net.Conn
		charts        *module.Charts
	}
)

var (
	// chronyCmdAddr is the chrony local port
	chronyDefaultProtocol = "udp"
	chronyDefaultCmdAddr  = "127.0.0.1:323"
)

func init() {
	creator := module.Creator{
		Create: func() module.Module { return New() },
	}

	module.Register("chrony", creator)
}

// New creates Chrony exposing local status of a chrony daemon
func New() *Chrony {
	return &Chrony{
		Config: Config{
			Protocol:     chronyDefaultProtocol,
			Address:      chronyDefaultCmdAddr,
			SentryConfig: sentry.ClientOptions{},
			Timeout:      500,
		},
		charts:       &charts,
		latestSource: net.IPv4zero,
	}
}

// Cleanup makes cleanup
func (c *Chrony) Cleanup() {
}

// Init makes initialization
func (c *Chrony) Init() bool {
	err := sentry.Init(c.SentryConfig)
	if err != nil {
		c.Warningf("Sentry initialization failed: %v", err)
	}
	if c.Timeout <= 0 {
		c.Timeout = 1000
	}

	conn, err := net.DialTimeout(c.Protocol, c.Address, time.Duration(c.Timeout)*time.Millisecond)
	if err != nil {
		c.Errorf(
			"unable connect to chrony addr %s:%s err: %s, is chrony up and running?",
			c.Protocol, c.Address, err)
		sentry.CaptureException(fmt.Errorf("connect chrony addr %s:%s err: %s",
			c.Protocol, c.Address, err))
		return false
	}

	c.conn = conn
	return c.ApplyChronyVersion() == nil
}

// Check makes check
func (c *Chrony) Check() bool {
	err := c.ApplyChronyVersion()
	if err != nil {
		c.Errorf("get chrony version failed with err: %s", err)
		sentry.CaptureException(
			fmt.Errorf("get chrony version failed with err: %s", err))
		return false
	}

	return true
}

// Charts creates Charts dynamically
func (c *Chrony) Charts() *Charts {
	return c.charts
}

// Collect collects metrics
func (c *Chrony) Collect() map[string]int64 {
	// collect all we need and sent Exception to sentry
	res := map[string]int64{"running": 0}
	var err error

	err = c.collectTracking(res)
	if err != nil {
		c.Errorf("fetch tracking status failed: %s", err)
	}

	err = c.collectActivity(res)
	if err != nil {
		c.Errorf("fetch activity status failed: %s", err)
	}

	return res
}

func (c *Chrony) collectTracking(res map[string]int64) error {
	tracking, err := c.FetchTracking()
	if err != nil {
		sentry.CaptureException((FetchingChronyError)(err.Error()))
	}
	c.Debugf(tracking.String())

	res["running"] = 1
	res["stratum"] = (int64)(tracking.Stratum)
	res["leap_status"] = (int64)(tracking.LeapStatus)
	res["root_delay"] = (int64)(tracking.RootDelay.Int64())
	res["root_dispersion"] = (int64)(tracking.RootDispersion.Int64())
	res["skew"] = (int64)(tracking.SkewPpm.Int64())
	res["frequency"] = (int64)(tracking.FreqPpm.Int64())
	res["last_offset"] = (int64)(tracking.LastOffset.Int64())
	res["rms_offset"] = (int64)(tracking.RmsOffset.Int64())
	res["update_interval"] = (int64)(tracking.LastUpdateInterval.Int64())
	res["current_correction"] = (int64)(tracking.CurrentCorrection.Int64())
	res["ref_timestamp"] = tracking.RefTime.Time().Unix()
	res["residual_frequency"] = (int64)(tracking.ResidFreqPpm.Int64())

	sourceIp := tracking.Ip.Ip()

	if !sourceIp.Equal(c.latestSource) {
		chart := c.charts.Get("source")
		_ = chart.AddDim(&module.Dim{
			ID: sourceIp.String(), Name: sourceIp.String(), Algo: module.Absolute, Div: 1, Mul: 1,
		})
		_ = chart.RemoveDim(c.latestSource.String())

		// you should let go.d.plugin know that something has been changed, and print dimension again.
		chart.MarkNotCreated()

		c.Debugf("source change from %s to %s")
		sentry.CaptureException(
			fmt.Errorf("source changed! {%s} -> {%s}", c.latestSource, sourceIp))
		c.latestSource = sourceIp
	}
	res[c.latestSource.String()] = 1

	if sourceIp.Equal(net.IPv4zero) || sourceIp.Equal(net.IPv6zero) {
		c.Debugf("sending sentry error for NoSourceOnlineError")
		sentry.CaptureException(NoSourceOnlineError(0))
	}

	// report root dispersion error to sentry when error > 100ms
	rd := tracking.RootDispersion.Float64()
	if rd > 0.1 {
		c.Debugf("sending sentry error for RootDispersionTooLargeError: %g", rd)
		sentry.CaptureException((RootDispersionTooLargeError)(rd))
	}

	// report frequency change to sentry when step > 500ppm
	fp := tracking.FreqPpm.Float64()
	if fp > 500 {
		c.Debugf("sending sentry error for FreqChangeTooFastError: %g", fp)
		sentry.CaptureException((FreqChangeTooFastError)(fp))
	}

	if tracking.LeapStatus != 0 {
		c.Debugf("sending sentry error for LeapStatusError: %g", tracking.LeapStatus)
		sentry.CaptureException((LeapStatusError)(tracking.LeapStatus))
	}

	rt := tracking.RefTime.Time()
	if time.Now().Sub(rt).Hours() >= 6 {
		c.Debugf("sending sentry error for TooLongNotSync: %s", rt.Format(time.RFC3339))
		sentry.CaptureException((OutOfSyncForTooLong)(rt))
	}

	return nil
}

func (c *Chrony) collectActivity(res map[string]int64) error {
	activity, err := c.FetchActivity()
	if err != nil {
		sentry.CaptureException((FetchingChronyError)(err.Error()))
	}

	res["online_sources"] = int64(activity.Online)
	res["offline_sources"] = int64(activity.Offline)
	res["burst_online_sources"] = int64(activity.BurstOnline)
	res["burst_offline_sources"] = int64(activity.BurstOffline)
	res["unresolved_sources"] = int64(activity.Unresolved)

	if activity.Online == 0 {
		c.Debugf("sending sentry error for NoSourceOnlineError: %g", activity.Online)
		sentry.CaptureException((NoSourceOnlineError)(activity.Online))
	}
	return nil
}
