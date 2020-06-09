package dmi

import (
	"github.com/VictorLowther/godmi"
	"strings"
)

type Processors struct {
	TotalCoreCount   uint32
	EnabledCoreCount uint32
	TotalThreadCount uint32
	Items            []*godmi.ProcessorInformation
}

type Memory struct {
	TotalCapacity  uint64
	Size           uint64
	TotalSlots     uint32
	PopulatedSlots uint32
	Arrays         []*godmi.PhysicalMemoryArray
	Devices        []*godmi.MemoryDevice
}

type Info struct {
	BIOS       *godmi.BIOSInformation
	System     *godmi.SystemInformation
	Baseboards []*godmi.BaseboardInformation
	Chassis    []*godmi.ChassisInformation
	Processors Processors
	Memory     Memory
	Hypervisor string
}

func (i *Info) Class() string {
	return "DMI"
}

func DetectVirtType(dmiinfo *Info) (string, bool) {
	keys := []string{dmiinfo.System.ProductName, dmiinfo.System.Manufacturer}
	for _, v := range dmiinfo.Baseboards {
		keys = append(keys, v.Manufacturer)
	}
	keys = append(keys, dmiinfo.BIOS.Vendor)
	vendors := [][2]string{
		{"KVM", "KVM"},
		{"QEMU", "QEMU"},
		{"VMware", "VMware"},
		{"VMW", "VMware"},
		{"innotek GmbH", "VirtualBox"},
		{"Oracle Corporation", "VirtualBox"},
		{"Xen", "Xen"},
		{"Bochs", "Bochs"},
		{"Parallels", "Parallels"},
		{"BHYVE", "BHYVE"},
	}
	for _, key := range keys {
		for _, vendor := range vendors {
			if strings.HasPrefix(key, vendor[0]) {
				return vendor[1], true
			}
		}
	}
	return "", false
}

func Gather() (res *Info, err error) {
	res = &Info{}
	if err = godmi.Init(); err != nil {
		return
	}
	// Filter out bad BIOS records
	for _, bios := range godmi.BIOSInformations {
		if bios.BIOSVersion == "" || bios.BIOSVersion == "Not Specified" {
			continue
		}
		if bios.ReleaseDate == "" || bios.ReleaseDate == "Not Specified" {
			continue
		}
		if bios.Vendor == "" || bios.Vendor == "Not Specified" {
			continue
		}
		res.BIOS = bios
		break
	}
	// filter out bad System records
	if len(godmi.SystemInformations) == 1 {
		res.System = godmi.SystemInformations[0]
	}
	res.Baseboards = godmi.BaseboardInformations
	res.Chassis = godmi.ChassisInformations
	res.Processors.Items = godmi.ProcessorInformations
	for _, proc := range res.Processors.Items {
		res.Processors.TotalCoreCount += uint32(proc.CoreCount)
		res.Processors.TotalThreadCount += uint32(proc.ThreadCount)
		res.Processors.EnabledCoreCount += uint32(proc.CoreEnabled)
	}
	res.Memory.Arrays = godmi.PhysicalMemoryArrays
	res.Memory.Devices = godmi.MemoryDevices
	for _, array := range res.Memory.Arrays {
		res.Memory.TotalCapacity += uint64(array.MaximumCapacity)
	}
	for _, device := range res.Memory.Devices {
		res.Memory.Size += device.Size
		res.Memory.TotalSlots += 1
		if device.Size != 0 {
			res.Memory.PopulatedSlots += 1
		}
	}
	res.Hypervisor,_ = DetectVirtType(res)
	return
}
