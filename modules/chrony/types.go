package chrony

import (
	"fmt"
	"math"
	"net"
	"time"
)

const (
	// protoVersionNumber is the protocol version for this client
	protoVersionNumber  = protoVersionNumber6
	protoVersionNumber6 = uint8(6)
	protoVersionNumber5 = uint8(5)

	// pktTypeCMDRequest is the request packet type
	pktTypeCMDRequest = uint8(1)
	// pktTypeCMDReply is the reply packet type
	pktTypeCMDReply = uint8(2)

	// reqTracking identifies a tracking request
	reqTracking = uint16(33)
	// reqActivity identifies an activity check request
	reqActivity = uint16(44)
	// reqNSources identifies a n_sources request
	reqNSources = uint16(14)
	// reqSourceStats identifies a sourcestats request
	reqSourceStats = uint16(34)
	// reqSourceData identifies a source data request
	reqSourceData = uint16(15)

	// rpyTracking identifies a tracking reply
	rpyTypeTracking = uint16(5)
	// rpyActivity identifies an activity check reply
	rpyTypeActivity = uint16(12)
	// reqNSources identifies a n_sources request
	rpyNSources = uint16(2)
	// reqSourcesStats identifies a sourcestats request
	rpySourcesStats = uint16(6)

	// floatExpBits represent 32-bit floating-point format consisting of 7-bit signed exponent
	floatExpBits = 7
	// floatCoefBits represent chronyFloat 25-bit signed coefficient without hidden bit
	floatCoefBits = 25
	// precision scaling factor
	scaleFactor = 1000000000
)

// RequestPacket holds a chrony request
type RequestPacket struct {
	Version   uint8 /* Protocol version */
	PktType   uint8 /* What sort of packet this is */
	Res1      uint8
	Res2      uint8
	Command   uint16 /* Which command is being issued */
	Attempt   uint16 /* How many resends the client has done (count up from zero for same sequence number) */
	SeqNumber uint32 /* Client's sequence number */
	Pad       [396]byte
}

// TrackingPayload is the payload for tracking replies (`RPY_Tracking`)
type TrackingPayload struct {
	RefID              uint32
	Ip                 IpAddr
	Stratum            uint16
	LeapStatus         uint16
	RefTime            chronyTimespec
	CurrentCorrection  chronyFloat
	LastOffset         chronyFloat
	RmsOffset          chronyFloat
	FreqPpm            chronyFloat
	ResidFreqPpm       chronyFloat
	SkewPpm            chronyFloat
	RootDelay          chronyFloat
	RootDispersion     chronyFloat
	LastUpdateInterval chronyFloat
}

const (
	IpaddrUnspec = uint16(0)
	IpaddrInet4  = uint16(1)
	IpaddrInet6  = uint16(2)
)

type IpAddr struct {
	IP       [16]uint8
	Family     uint16
	Pad        uint16
}

func (tracking *TrackingPayload) String() string {
	return fmt.Sprintf(
		"RefID: %d, ActivictServer: %s, Stratum: %d, RefTime: %s, CurrentCorrection: %f, "+
			"FreqPpm: %f, SkewPpm: %f, RootDelay: %f, "+
			"RootDispersion: %f, LeapStatus: %d, LastUpdateInterval: %f, "+
			"LastOffset: %f, CurrentCorrection: %f",
		tracking.RefID, tracking.Ip.String(), tracking.Stratum, tracking.RefTime.Time().Format(time.RFC3339),
		tracking.CurrentCorrection.Float64(), tracking.FreqPpm.Float64(), tracking.SkewPpm.Float64(),
		tracking.RootDelay.Float64(), tracking.RootDispersion.Float64(), tracking.LeapStatus,
		tracking.LastUpdateInterval.Float64(), tracking.LastOffset.Float64(), tracking.CurrentCorrection.Float64(),
	)
}

func (ia IpAddr) Ip() net.IP {
	if ia.Family == IpaddrInet4 {
		return net.IP(ia.IP[:4])
	}
	return net.IP(ia.IP[:])
}

func (ia IpAddr) String() string {
	return ia.Ip().String()
}

// ActivityPayload is the payload for activity replies (`RPY_Activity`)
type ActivityPayload struct {
	Online       int32
	Offline      int32
	BurstOnline  int32
	BurstOffline int32
	Unresolved   int32
}

func (activity *ActivityPayload) String() string {
	return fmt.Sprintf("Online: %d, Offline: %d, BurstOnline: %d, BurstOffline: %d, Unresolved: %d",
		activity.Online, activity.Offline, activity.BurstOnline, activity.BurstOffline, activity.Unresolved,
	)
}

// ReplyPacket is the common header for all replies
// chrony version 4.1
type ReplyPacket struct {
	Version uint8
	PktType uint8
	Res1    uint8
	Res2    uint8
	Command uint16
	Reply   uint16
	Status  uint16
	Pad1    uint16
	Pad2    uint16
	Pad3    uint16
	SeqNum  uint32
	Pad4    uint32
	Pad5    uint32
}

// chronyFloat is the custom chrony timespec type (`Timespec`)
type chronyTimespec struct {
	TvSecHigh uint32
	TvSecLow  uint32
	TvNSec    uint32
}

func (ct chronyTimespec) Time() time.Time {
	var nsec = uint32(999999999)
	if ct.TvNSec < nsec {
		nsec = ct.TvNSec
	}

	return time.Unix(int64(uint64(ct.TvSecHigh)<<32+uint64(ct.TvSecLow)), int64(nsec))
}

// EpochSeconds returns the number of seconds since epoch
func (ct chronyTimespec) EpochSeconds() float64 {
	ts := uint64(ct.TvSecHigh) << 32
	ts += uint64(ct.TvSecLow)
	return float64(ts)
}


// chronyFloat is a 32-bit floating-point 
// format consisting of 7-bit signed exponent 
// and 25-bit signed coefficient without hidden bit.
// The result is calculated as: 2^(exp - 25) * coef
type chronyFloat int32

// ToFloat does magic to decode float from int32.
// Code is copied and translated to Go from original C sources.
func (f chronyFloat) Float64() float64 {
	var exp, coef int32

	x := uint32(f)

	exp = int32(x >> floatCoefBits)
	if exp >= 1<<(floatExpBits-1) {
		exp -= 1 << floatExpBits
	}
	exp -= floatCoefBits

	coef = int32(x % (1 << floatCoefBits))
	if coef >= 1<<(floatCoefBits-1) {
		coef -= 1 << floatCoefBits
	}

	return float64(coef) * math.Pow(2.0, float64(exp))
}

// Int64 returns the 64bits float value
func (cf chronyFloat) Int64() int64 {
	return int64(cf.Float64() * scaleFactor)
}

// RootDispersionTooLargeError
type RootDispersionTooLargeError float64

func (f RootDispersionTooLargeError) Error() string {
	return fmt.Sprintf("root dispersion too large: %g", float64(f))
}

// FreqChangeTooFastError
type FreqChangeTooFastError float64

func (f FreqChangeTooFastError) Error() string {
	return fmt.Sprintf("chrony frequency change too fast: %g", float64(f))
}

// LeapStatusError
type LeapStatusError float64

func (f LeapStatusError) Error() string {
	return fmt.Sprintf("chrony abnormal leap status: %g", float64(f))
}

// FetchingChronyError
type FetchingChronyError string

func (f FetchingChronyError) Error() string {
	return fmt.Sprintf("can't read from chrony socket: %s", string(f))
}

// TooLongNotSync
type OutOfSyncForTooLong time.Time

func (e OutOfSyncForTooLong) Error() string {
	t := time.Time(e)
	return fmt.Sprintf("last update at: %s, %g min ago",
		t.Format(time.RFC3339), time.Now().Sub(t).Minutes())
}

// NoSourceOnlineError
type NoSourceOnlineError int

func (e NoSourceOnlineError) Error() string {
	return "no source online"
}
