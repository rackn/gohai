// +build ppc64le
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
	var vendorId string
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
		case "cpu":
			proc.Model = v
		case "clock":
			proc.Speed = v

		case "model":
			vendorId = v
		}
	}
	for ii, _ := range i.Processors {
		i.Processors[ii].Vendor = vendorId
		i.Processors[ii].Cores = 1

	}
	return nil
}
