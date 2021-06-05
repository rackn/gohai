package system

import (
	"runtime"
	"strconv"
)

type Processor struct {
	ID             int64
	Vendor         string
	Family         int64
	ModelCode      int64
	Model          string
	Stepping       int64
	Microcode      int64
	Speed          string
	CacheSize      string
	PhysID         int64
	Sibligs        int64
	CoreID         int64
	Cores          int64
	FPU            bool
	WriteProtect   bool
	Flags          []string
	Bugs           []string
	CacheAlignment int64
	AddressSizes   struct {
		Physical int64
		Virtual  int64
	}
}

type Info struct {
	OS     string
	Arch   string
	Kernel string
	Memory struct {
		Total     int64
		Free      int64
		Available int64
	}
	ProcessorCount int
	Processors     []Processor
}

func (i *Info) Class() string {
	return "System"
}

func mPI(s string, size int) int64 {
	res, err := strconv.ParseInt(s, 0, size)
	if err != nil {
		panic("Failed to parse int")
	}
	return res
}

func Gather() (*Info, error) {
	res := &Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	switch res.OS {
	case "linux":
		return res, fillLinux(res)
	}
	return res, nil
}
