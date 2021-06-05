// +build !ppc64le

package system

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func fillLinux(i *Info) error {
	vbytes, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return err
	}
	fields := bytes.Split(vbytes, []byte(" "))
	i.Kernel = string(fields[2])
	memInfo, err := os.Open("/proc/meminfo")
	if err != nil {
		return err
	}
	defer memInfo.Close()
	lines := bufio.NewScanner(memInfo)
	for lines.Scan() {
		frags := strings.SplitN(lines.Text(), ":", 2)
		szPart := strings.Split(strings.TrimSpace(frags[1]), " ")[0]
		sz, err := strconv.ParseInt(szPart, 10, 64)
		if err != nil {
			return err
		}
		switch frags[0] {
		case "MemTotal":
			i.Memory.Total = sz << 10
		case "MemFree":
			i.Memory.Free = sz << 10
		case "MemAvailable":
			i.Memory.Available = sz << 10
		default:
			break
		}
	}
	cpuInfo, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return err
	}
	defer cpuInfo.Close()
	i.Processors = []Processor{}
	lines = bufio.NewScanner(cpuInfo)
	var proc Processor
	for lines.Scan() {
		frags := strings.SplitN(lines.Text(), ":", 2)
		if len(frags) != 2 {
			i.Processors = append(i.Processors, proc)
			continue
		}
		k, v := strings.TrimSpace(frags[0]), strings.TrimSpace(frags[1])

		switch k {
		case "processor":
			proc = Processor{}
			proc.ID = mPI(v, 64)
			i.ProcessorCount += 1
		case "vendor_id":
			proc.Vendor = v
		case "cpu family":
			proc.Family = mPI(v, 64)
		case "model":
			proc.ModelCode = mPI(v, 64)
		case "model name":
			proc.Model = v
		case "stepping":
			proc.Stepping = mPI(v, 64)
		case "microcode":
			proc.Microcode = mPI(v, 64)
		case "cpu MHz":
			proc.Speed = v
		case "cache size":
			proc.CacheSize = v
		case "physical id":
			proc.PhysID = mPI(v, 64)
		case "siblings":
			proc.Sibligs = mPI(v, 64)
		case "core id":
			proc.CoreID = mPI(v, 64)
		case "cpu cores":
			proc.Cores = mPI(v, 64)
		case "fpu":
			proc.FPU = v == "yes"
		case "wp":
			proc.WriteProtect = v == "yes"
		case "flags":
			proc.Flags = strings.Split(v, " ")
		case "bugs":
			if len(v) == 0 {
				proc.Bugs = []string{}
			} else {
				proc.Bugs = strings.Split(v, " ")
			}
		case "cache_alignment":
			proc.CacheAlignment = mPI(v, 64)
		case "address sizes":
			aParts := strings.Split(v, " ")
			proc.AddressSizes.Physical = mPI(aParts[0], 64)
			proc.AddressSizes.Virtual = mPI(aParts[3], 64)
		}
	}
	return nil
}
