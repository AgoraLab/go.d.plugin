//go:build !(linux && cgo && (amd64 || ppc64le || arm64))
// +build !linux !cgo !amd64,!ppc64le,!arm64

package nvidia_smi

import (
	"github.com/netdata/go.d.plugin/agent/module"
	"runtime"
)

func init() {
	creator := module.Creator{
		Create: func() module.Module { return New() },
	}

	module.Register("nvidia_smi", creator)
}

// Nvsmi module struct
type Nvsmi struct {
	module.Base // should be embedded by every module
	metrics     map[string]int64
}

func New() *Nvsmi {
	return &Nvsmi{
		metrics: make(map[string]int64),
	}
}

func (n *Nvsmi) Init() bool {
	n.Error("nvidia smi not support for", runtime.GOOS, runtime.GOARCH)
	return true
}

func (n *Nvsmi) Check() bool {
	n.Error("nvidia smi not support for", runtime.GOOS, runtime.GOARCH)
	return true
}

func (n *Nvsmi) Charts() *module.Charts {
	n.Error("nvidia smi not support for", runtime.GOOS, runtime.GOARCH)
	return &Charts{}
}

func (n *Nvsmi) Collect() map[string]int64 {
	n.Error("nvidia smi not support for", runtime.GOOS, runtime.GOARCH)
	return nil
}

func (Nvsmi) Cleanup() {}
