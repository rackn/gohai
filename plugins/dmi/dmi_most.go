// +build !ppc64le

package dmi

import (
	"strings"

	"github.com/VictorLowther/godmi"
)

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
	return processDMI()
}
