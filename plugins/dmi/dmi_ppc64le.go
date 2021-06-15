// +build ppc64le

package dmi

import (
	"encoding/json"
	"os/exec"
	"strconv"

	"github.com/VictorLowther/godmi"
)

func DetectVirtType(dmiinfo *Info) (string, bool) {
	return "LPAR", true
}

func getStringFromMap(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

// choose first non-empty string, else empty
func chooseNonEmpty(list ...string) string {
	for _, s := range list {
		if s != "" {
			return s
		}
	}
	return ""
}

func Gather() (res *Info, err error) {
	// Just in case they have DMI, use it
	if gerr := godmi.Init(); gerr == nil {
		return processDMI()
	}

	var jsonOut []byte
	jsonOut, err = exec.Command("lshw", "-json").Output()
	if err != nil {
		return
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonOut, &result)
	if err != nil {
		return
	}

	/* Example json blob
	"id" : "p362n01.pbm.ihost.com",
	"class" : "system",
	"claimed" : true,
	"description" : "pSeries LPAR",
	"product" : "IBM,8247-22L",
	"vendor" : "IBM",
	"serial" : "IBM,03212169A",
	"width" : 64,
	"capabilities" : {
	  "smp" : "Symmetric Multi-Processing"
	},
	*/

	var core map[string]interface{}
	var firmware map[string]interface{}
	rescpu := []map[string]interface{}{}
	resmem := []map[string]interface{}{}

	children := result["children"].([]interface{})
	for _, obj := range children {
		c := obj.(map[string]interface{})
		id := getStringFromMap(c, "id")
		if id == "core" {
			core = c
		}
	}

	children = core["children"].([]interface{})
	for _, obj := range children {
		c := obj.(map[string]interface{})
		id := getStringFromMap(c, "id")
		class := getStringFromMap(c, "class")
		if id == "firmware" {
			firmware = c
		} else if class == "memory" {
			resmem = append(resmem, c)
		} else if class == "processor" {
			rescpu = append(rescpu, c)
		}
	}

	// Make up stuff from lshw and other stuff.
	res = &Info{}
	b := &godmi.BIOSInformation{
		Vendor: getStringFromMap(result, "vendor"),
		BIOSVersion: chooseNonEmpty(
			getStringFromMap(firmware, "version"),
			getStringFromMap(result, "version"),
			getStringFromMap(result, "product")),
		ReleaseDate: chooseNonEmpty(
			getStringFromMap(firmware, "date"),
			getStringFromMap(firmware, "version"),
			getStringFromMap(result, "version"),
			getStringFromMap(result, "product")),
		SystemBIOSMajorRelease:                 0,
		SystemBIOSMinorRelease:                 0,
		EmbeddedControllerFirmwareMajorRelease: 0,
		EmbeddedControllerFirmawreMinorRelease: 0,
	}
	res.BIOS = b

	s := &godmi.SystemInformation{
		Manufacturer: getStringFromMap(result, "vendor"),
		ProductName:  getStringFromMap(result, "product"),
		Version:      chooseNonEmpty(getStringFromMap(result, "version"), getStringFromMap(result, "product")),
		SerialNumber: getStringFromMap(result, "serial"),
		Family:       getStringFromMap(result, "description"),
	}
	res.System = s

	bs := []*godmi.BaseboardInformation{
		&godmi.BaseboardInformation{
			Manufacturer: getStringFromMap(result, "vendor"),
			ProductName:  getStringFromMap(result, "product"),
			Version:      chooseNonEmpty(getStringFromMap(result, "version"), getStringFromMap(result, "product")),
			SerialNumber: getStringFromMap(result, "serial"),
			BoardType:    10,
		},
	}
	res.Baseboards = bs
	res.Chassis = []*godmi.ChassisInformation{}

	res.Processors.Items = godmi.ProcessorInformations
	/* Processor example
			    "id" : "cpu:1",
				"class" : "processor",
				"claimed" : true,
				"description" : "POWER8 (architected), altivec supported",
				"product" : "PowerPC,POWER8",
				"physid" : "16",
				"businfo" : "cpu@1",
				"version" : "2.1 (pvr 004b 0201)",
				"units" : "Hz",
				"size" : 3026000000,
				"configuration" : {
	     			"threads" : "8"
		    	},

	          "id" : "cpu",
	          "class" : "processor",
	          "claimed" : true,
	          "product" : "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz",
	          "vendor" : "Intel Corp.",
	          "physid" : "2",
	          "businfo" : "cpu@0",
	          "version" : "6.158.13",
	          "width" : 64,
	*/
	for ii, p := range rescpu {
		t := byte(1)
		if d, ok := p["configuration"]; ok {
			if mp, ok := d.(map[string]interface{}); ok {
				if tv, ok := mp["threads"].(string); ok {
					i, e := strconv.Atoi(tv)
					if e == nil {
						t = byte(i)
					}
				}
			}
		}
		s := uint16(0)
		if sv, ok := p["size"].(string); ok {
			i, e := strconv.ParseUint(sv, 10, 64)
			if e == nil {
				s = uint16(i / 1000000)
			}
		}
		np := &godmi.ProcessorInformation{
			SocketDesignation: chooseNonEmpty(getStringFromMap(p, "product")),
			ProcessorType:     3,
			Family:            godmi.ProcessorPowerPCFamily,
			Manufacturer:      chooseNonEmpty(getStringFromMap(p, "vendor"), getStringFromMap(firmware, "vendor"), getStringFromMap(result, "vendor")),
			ID:                godmi.ProcessorID(ii),
			Version:           getStringFromMap(p, "version"),
			MaxSpeed:          s,
			CurrentSpeed:      s,
			CoreCount:         1,
			CoreEnabled:       1,
			ThreadCount:       t,
		}
		res.Processors.Items = append(res.Processors.Items, np)
	}
	res.Memory.Arrays = []*godmi.PhysicalMemoryArray{
		&godmi.PhysicalMemoryArray{
			Location:              0,
			Use:                   3,
			ErrorCorrection:       3,
			MaximumCapacity:       0,
			NumberOfMemoryDevices: uint16(len(resmem)),
		},
	}

	/*
	   {
	     "id" : "memory",
	     "class" : "memory",
	     "claimed" : true,
	     "description" : "System memory",
	     "physid" : "2",
	     "units" : "bytes",
	     "size" : 34359738368
	   }
	*/
	res.Memory.Devices = godmi.MemoryDevices
	for _, m := range resmem {
		size, _ := m["size"].(float64)
		nm := &godmi.MemoryDevice{
			Size: uint64(size),
		}
		res.Memory.Devices = append(res.Memory.Devices, nm)
		res.Memory.Arrays[0].MaximumCapacity += uint64(size)
	}

	for _, proc := range res.Processors.Items {
		res.Processors.TotalCoreCount += uint32(proc.CoreCount)
		res.Processors.TotalThreadCount += uint32(proc.ThreadCount)
		res.Processors.EnabledCoreCount += uint32(proc.CoreEnabled)
	}
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
	res.Hypervisor, _ = DetectVirtType(res)
	return
}
