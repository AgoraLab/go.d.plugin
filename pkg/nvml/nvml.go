//go:build linux && cgo && (amd64 || ppc64le || arm64)
// +build linux
// +build cgo
// +build amd64 ppc64le arm64

// Copyright (c) 2015-2018, NVIDIA CORPORATION. All rights reserved.

package nvml

/*
// Copyright (c) 2015-2018, NVIDIA CORPORATION. All rights reserved.

#include <stddef.h>
#include <dlfcn.h>

#include "nvml_dl.h"

#define DLSYM(x, sym)                           \
do {                                            \
    dlerror();				        \
    x = dlsym(handle, #sym);                    \
    if (dlerror() != NULL) {                    \
        return (NVML_ERROR_FUNCTION_NOT_FOUND); \
    }                                           \
} while (0)

typedef nvmlReturn_t (*nvmlSym_t)();

static void *handle;

nvmlReturn_t NVML_DL(nvmlInit)(void)
{
    handle = dlopen("libnvidia-ml.so.1", RTLD_LAZY | RTLD_GLOBAL);
    if (handle == NULL) {
	return (NVML_ERROR_LIBRARY_NOT_FOUND);
    }
    return (nvmlInit());
}

nvmlReturn_t NVML_DL(nvmlShutdown)(void)
{
    nvmlReturn_t r = nvmlShutdown();
    if (r != NVML_SUCCESS) {
	return (r);
    }
    return (dlclose(handle) ? NVML_ERROR_UNKNOWN : NVML_SUCCESS);
}

nvmlReturn_t NVML_DL(nvmlDeviceGetTopologyCommonAncestor)(
  nvmlDevice_t dev1, nvmlDevice_t dev2, nvmlGpuTopologyLevel_t *info)
{
    nvmlSym_t sym;

    DLSYM(sym, nvmlDeviceGetTopologyCommonAncestor);
    return ((*sym)(dev1, dev2, info));
}
*/
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

var (
	ErrCPUAffinity        = errors.New("failed to retrieve CPU affinity")
	ErrUnsupportedP2PLink = errors.New("unsupported P2P link type")
	ErrUnsupportedGPU     = errors.New("unsupported GPU device")
)

type ModeState uint

const (
	Disabled ModeState = iota
	Enabled
)

func (m ModeState) String() string {
	switch m {
	case Enabled:
		return "Enabled"
	case Disabled:
		return "Disabled"
	}
	return "N/A"
}

type Display struct {
	Mode   ModeState
	Active ModeState
}

type Accounting struct {
	Mode       ModeState
	BufferSize *uint
}

type DeviceMode struct {
	DisplayInfo    Display
	Persistence    ModeState
	AccountingInfo Accounting
}

type ThrottleReason uint

const (
	ThrottleReasonGpuIdle ThrottleReason = iota
	ThrottleReasonApplicationsClocksSetting
	ThrottleReasonSwPowerCap
	ThrottleReasonHwSlowdown
	ThrottleReasonSyncBoost
	ThrottleReasonSwThermalSlowdown
	ThrottleReasonHwThermalSlowdown
	ThrottleReasonHwPowerBrakeSlowdown
	ThrottleReasonDisplayClockSetting
	ThrottleReasonNone
	ThrottleReasonUnknown
)

func (r ThrottleReason) String() string {
	switch r {
	case ThrottleReasonGpuIdle:
		return "Gpu Idle"
	case ThrottleReasonApplicationsClocksSetting:
		return "Applications Clocks Setting"
	case ThrottleReasonSwPowerCap:
		return "SW Power Cap"
	case ThrottleReasonHwSlowdown:
		return "HW Slowdown"
	case ThrottleReasonSyncBoost:
		return "Sync Boost"
	case ThrottleReasonSwThermalSlowdown:
		return "SW Thermal Slowdown"
	case ThrottleReasonHwThermalSlowdown:
		return "HW Thermal Slowdown"
	case ThrottleReasonHwPowerBrakeSlowdown:
		return "HW Power Brake Slowdown"
	case ThrottleReasonDisplayClockSetting:
		return "Display Clock Setting"
	case ThrottleReasonNone:
		return "No clocks throttling"
	}
	return "N/A"
}

type PerfState uint

const (
	PerfStateMax     = 0
	PerfStateMin     = 15
	PerfStateUnknown = 32
)

func (p PerfState) String() string {
	if p >= PerfStateMax && p <= PerfStateMin {
		return fmt.Sprintf("P%d", p)
	}
	return "Unknown"
}

type ProcessType uint

const (
	Compute ProcessType = iota
	Graphics
	ComputeAndGraphics
)

func (t ProcessType) String() string {
	typ := "C+G"
	if t == Compute {
		typ = "C"
	} else if t == Graphics {
		typ = "G"
	}
	return typ
}

type P2PLinkType uint

const (
	P2PLinkUnknown P2PLinkType = iota
	P2PLinkCrossCPU
	P2PLinkSameCPU
	P2PLinkHostBridge
	P2PLinkMultiSwitch
	P2PLinkSingleSwitch
	P2PLinkSameBoard
	SingleNVLINKLink
	TwoNVLINKLinks
	ThreeNVLINKLinks
	FourNVLINKLinks
	FiveNVLINKLinks
	SixNVLINKLinks
)

type P2PLink struct {
	BusID string
	Link  P2PLinkType
}

func (t P2PLinkType) String() string {
	switch t {
	case P2PLinkCrossCPU:
		return "Cross CPU socket"
	case P2PLinkSameCPU:
		return "Same CPU socket"
	case P2PLinkHostBridge:
		return "Host PCI bridge"
	case P2PLinkMultiSwitch:
		return "Multiple PCI switches"
	case P2PLinkSingleSwitch:
		return "Single PCI switch"
	case P2PLinkSameBoard:
		return "Same board"
	case SingleNVLINKLink:
		return "Single NVLink"
	case TwoNVLINKLinks:
		return "Two NVLinks"
	case ThreeNVLINKLinks:
		return "Three NVLinks"
	case FourNVLINKLinks:
		return "Four NVLinks"
	case FiveNVLINKLinks:
		return "Five NVLinks"
	case SixNVLINKLinks:
		return "Six NVLinks"
	case P2PLinkUnknown:
	}
	return "N/A"
}

type ClockInfo struct {
	Cores  *uint
	Memory *uint
}

type PCIInfo struct {
	BusID     string
	BAR1      *uint64
	Bandwidth *uint
}

type Device struct {
	handle

	UUID        string
	Path        string
	Model       *string
	Power       *uint
	Memory      *uint64
	CPUAffinity *uint
	PCI         PCIInfo
	Clocks      ClockInfo
	Topology    []P2PLink
}

type UtilizationInfo struct {
	GPU     *uint
	Memory  *uint
	Encoder *uint
	Decoder *uint
}

type PCIThroughputInfo struct {
	RX *uint
	TX *uint
}

type PCIStatusInfo struct {
	BAR1Used   *uint64
	Throughput PCIThroughputInfo
}

type ECCErrorsInfo struct {
	L1Cache *uint64
	L2Cache *uint64
	Device  *uint64
}

type DeviceMemory struct {
	Used *uint64
	Free *uint64
}

type MemoryInfo struct {
	Global    DeviceMemory
	ECCErrors ECCErrorsInfo
}

type ProcessInfo struct {
	PID        uint
	Name       string
	MemoryUsed uint64
	Type       ProcessType
}

type DeviceStatus struct {
	Power       *uint
	Temperature *uint
	Utilization UtilizationInfo
	Memory      MemoryInfo
	Clocks      ClockInfo
	PCI         PCIStatusInfo
	Processes   []ProcessInfo
	Throttle    ThrottleReason
	Performance PerfState
}

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func Init() error {
	return init_()
}

func Shutdown() error {
	return shutdown()
}

func GetDeviceCount() (uint, error) {
	return deviceGetCount()
}

func GetDriverVersion() (string, error) {
	return systemGetDriverVersion()
}

func GetCudaDriverVersion() (*uint, *uint, error) {
	return systemGetCudaDriverVersion()
}

func numaNode(busid string) (uint, error) {
	// discard leading zeros of busid
	b, err := ioutil.ReadFile(fmt.Sprintf("/sys/bus/pci/devices/%s/numa_node", strings.ToLower(busid[4:])))
	if err != nil {
		// XXX report node 0 if NUMA support isn't enabled
		return 0, nil
	}
	node, err := strconv.ParseInt(string(bytes.TrimSpace(b)), 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%v: %v", ErrCPUAffinity, err)
	}
	if node < 0 {
		node = 0 // XXX report node 0 instead of NUMA_NO_NODE
	}
	return uint(node), nil
}

func pciBandwidth(gen, width *uint) *uint {
	m := map[uint]uint{
		1: 250, // MB/s
		2: 500,
		3: 985,
		4: 1969,
	}
	if gen == nil || width == nil {
		return nil
	}
	bw := m[*gen] * *width
	return &bw
}

func NewDevice(idx uint) (device *Device, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	h, err := deviceGetHandleByIndex(idx)
	assert(err)
	model, err := h.deviceGetName()
	assert(err)
	uuid, err := h.deviceGetUUID()
	assert(err)
	minor, err := h.deviceGetMinorNumber()
	assert(err)
	power, err := h.deviceGetPowerManagementLimit()
	assert(err)
	totalMem, _, err := h.deviceGetMemoryInfo()
	assert(err)
	busid, err := h.deviceGetPciInfo()
	assert(err)
	bar1, _, err := h.deviceGetBAR1MemoryInfo()
	assert(err)
	pcig, err := h.deviceGetMaxPcieLinkGeneration()
	assert(err)
	pciw, err := h.deviceGetMaxPcieLinkWidth()
	assert(err)
	ccore, cmem, err := h.deviceGetMaxClockInfo()
	assert(err)

	if minor == nil || busid == nil || uuid == nil {
		return nil, ErrUnsupportedGPU
	}
	path := fmt.Sprintf("/dev/nvidia%d", *minor)
	node, err := numaNode(*busid)
	assert(err)

	device = &Device{
		handle:      h,
		UUID:        *uuid,
		Path:        path,
		Model:       model,
		Power:       power,
		Memory:      totalMem,
		CPUAffinity: &node,
		PCI: PCIInfo{
			BusID:     *busid,
			BAR1:      bar1,
			Bandwidth: pciBandwidth(pcig, pciw), // MB/s
		},
		Clocks: ClockInfo{
			Cores:  ccore, // MHz
			Memory: cmem,  // MHz
		},
	}
	if power != nil {
		*device.Power /= 1000 // W
	}
	if bar1 != nil {
		*device.PCI.BAR1 /= 1024 * 1024 // MiB
	}
	return
}

func NewDeviceLite(idx uint) (device *Device, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	h, err := deviceGetHandleByIndex(idx)
	assert(err)
	uuid, err := h.deviceGetUUID()
	assert(err)
	minor, err := h.deviceGetMinorNumber()
	assert(err)
	busid, err := h.deviceGetPciInfo()
	assert(err)

	if minor == nil || busid == nil || uuid == nil {
		return nil, ErrUnsupportedGPU
	}
	path := fmt.Sprintf("/dev/nvidia%d", *minor)

	device = &Device{
		handle: h,
		UUID:   *uuid,
		Path:   path,
		PCI: PCIInfo{
			BusID: *busid,
		},
	}
	return
}

func (d *Device) Status() (status *DeviceStatus, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	power, err := d.deviceGetPowerUsage()
	assert(err)
	temp, err := d.deviceGetTemperature()
	assert(err)
	ugpu, umem, err := d.deviceGetUtilizationRates()
	assert(err)
	uenc, err := d.deviceGetEncoderUtilization()
	assert(err)
	udec, err := d.deviceGetDecoderUtilization()
	assert(err)
	_, devMem, err := d.deviceGetMemoryInfo()
	assert(err)
	ccore, cmem, err := d.deviceGetClockInfo()
	assert(err)
	_, bar1, err := d.deviceGetBAR1MemoryInfo()
	assert(err)
	el1, el2, emem, err := d.deviceGetMemoryErrorCounter()
	assert(err)
	pcirx, pcitx, err := d.deviceGetPcieThroughput()
	assert(err)
	throttle, err := d.getClocksThrottleReasons()
	assert(err)
	perfState, err := d.getPerformanceState()
	assert(err)
	processInfo, err := d.deviceGetAllRunningProcesses()
	assert(err)

	status = &DeviceStatus{
		Power:       power,
		Temperature: temp, // °C
		Utilization: UtilizationInfo{
			GPU:     ugpu, // %
			Memory:  umem, // %
			Encoder: uenc, // %
			Decoder: udec, // %
		},
		Memory: MemoryInfo{
			Global: devMem,
			ECCErrors: ECCErrorsInfo{
				L1Cache: el1,
				L2Cache: el2,
				Device:  emem,
			},
		},
		Clocks: ClockInfo{
			Cores:  ccore, // MHz
			Memory: cmem,  // MHz
		},
		PCI: PCIStatusInfo{
			BAR1Used: bar1,
			Throughput: PCIThroughputInfo{
				RX: pcirx,
				TX: pcitx,
			},
		},
		Throttle:    throttle,
		Performance: perfState,
		Processes:   processInfo,
	}
	if power != nil {
		*status.Power /= 1000 // W
	}
	if bar1 != nil {
		*status.PCI.BAR1Used /= 1024 * 1024 // MiB
	}
	if pcirx != nil {
		*status.PCI.Throughput.RX /= 1000 // MB/s
	}
	if pcitx != nil {
		*status.PCI.Throughput.TX /= 1000 // MB/s
	}
	return
}

func GetP2PLink(dev1, dev2 *Device) (link P2PLinkType, err error) {
	level, err := deviceGetTopologyCommonAncestor(dev1.handle, dev2.handle)
	if err != nil || level == nil {
		return P2PLinkUnknown, err
	}

	switch *level {
	case C.NVML_TOPOLOGY_INTERNAL:
		link = P2PLinkSameBoard
	case C.NVML_TOPOLOGY_SINGLE:
		link = P2PLinkSingleSwitch
	case C.NVML_TOPOLOGY_MULTIPLE:
		link = P2PLinkMultiSwitch
	case C.NVML_TOPOLOGY_HOSTBRIDGE:
		link = P2PLinkHostBridge
	case C.NVML_TOPOLOGY_CPU:
		link = P2PLinkSameCPU
	case C.NVML_TOPOLOGY_SYSTEM:
		link = P2PLinkCrossCPU
	default:
		err = ErrUnsupportedP2PLink
	}
	return
}

func GetNVLink(dev1, dev2 *Device) (link P2PLinkType, err error) {
	nvbusIds1, err := dev1.handle.deviceGetAllNvLinkRemotePciInfo()
	if err != nil || nvbusIds1 == nil {
		return P2PLinkUnknown, err
	}

	nvlink := P2PLinkUnknown
	for _, nvbusId1 := range nvbusIds1 {
		if *nvbusId1 == dev2.PCI.BusID {
			switch nvlink {
			case P2PLinkUnknown:
				nvlink = SingleNVLINKLink
			case SingleNVLINKLink:
				nvlink = TwoNVLINKLinks
			case TwoNVLINKLinks:
				nvlink = ThreeNVLINKLinks
			case ThreeNVLINKLinks:
				nvlink = FourNVLINKLinks
			case FourNVLINKLinks:
				nvlink = FiveNVLINKLinks
			case FiveNVLINKLinks:
				nvlink = SixNVLINKLinks
			}
		}
	}

	// TODO(klueska): Handle NVSwitch semantics

	return nvlink, nil
}

func (d *Device) GetComputeRunningProcesses() ([]uint, []uint64, error) {
	return d.handle.deviceGetComputeRunningProcesses()
}

func (d *Device) GetGraphicsRunningProcesses() ([]uint, []uint64, error) {
	return d.handle.deviceGetGraphicsRunningProcesses()
}

func (d *Device) GetAllRunningProcesses() ([]ProcessInfo, error) {
	return d.handle.deviceGetAllRunningProcesses()
}

func (d *Device) GetDeviceMode() (mode *DeviceMode, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	display, err := d.getDisplayInfo()
	assert(err)

	p, err := d.getPeristenceMode()
	assert(err)

	accounting, err := d.getAccountingInfo()
	assert(err)

	mode = &DeviceMode{
		DisplayInfo:    display,
		Persistence:    p,
		AccountingInfo: accounting,
	}
	return
}
