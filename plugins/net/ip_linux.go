//+build linux

package net

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type ifReq struct {
	ifName [16]byte
	data   uintptr
}

func (i *ifReq) Name() string {
	return string(i.ifName[:])
}

func (i *ifReq) SetName(name string) {
	copy(i.ifName[:], name)
	i.ifName[15] = 0
}

func (i *ifReq) ioctl(cmd uint32, buf []byte) error {
	endian.PutUint32(buf[:4], cmd)
	bufHdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	i.data = bufHdr.Data
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_IP)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	_, _, errCode := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		SIOCETHTOOL,
		uintptr(unsafe.Pointer(i)))
	if errCode != 0 {
		return syscall.Errno(errCode)
	}
	return nil
}

/* buf layout for GSET:
0..3: cmd
4..7: supported features
8..11: advertised features
12..13: low bits of speed
14: duplex
15: port in use
16: MDIO phy address
17: transceiver to use
18: autonegotiation
19: MDIO support
20..23: max tx packets before an interrupt
24..27: max rx packets before an interrupt
28..29: high bits of speed
30: tp mdix
31: reserved
32..35: partner advertised features
36..43: reserved
*/
func (i *Interface) fillGset(buf []byte) error {
	speedLo := endian.Uint16(buf[12:14])
	speedHi := endian.Uint16(buf[28:30])
	i.Speed = (uint32(speedHi) << 16) + uint32(speedLo)
	i.Duplex = buf[14] != 0
	i.Autonegotiation = buf[18] != 0
	i.Supported = toModeBits(buf[4:8])
	i.Advertised = toModeBits(buf[8:12])
	i.PeerAdvertised = toModeBits(buf[32:36])
	return nil
}

/* buf layout for GLINKSETTINGS:
0..3: cmd
4..7: speed
8: duplex
9: port
10: phy address
11: autonegotiation
12: MDIO support
13: eth tp mdix
14: eth tp mdix control
15: number of 32 bit words to be used for the
    supported features, advertised features, and peer advertised features bits
16..47
48: supported features, advertized features, peer advertised features
*/

func (i *Interface) fillGlink(buf []byte) error {
	i.Speed = endian.Uint32(buf[4:8])
	i.Duplex = buf[8] != 0
	i.Autonegotiation = buf[11] != 0
	b := int(buf[15]) << 2
	s := 48
	a := 48 + b
	p := a + b
	i.Supported = toModeBits(buf[s : s+b])
	i.Advertised = toModeBits(buf[a : a+b])
	i.PeerAdvertised = toModeBits(buf[p : p+b])
	return nil
}

func (i *Interface) fillUdev() error {
	cmd := exec.Command("udevadm", "info", "-q", "all", "-p", "/sys/class/net/"+i.Name)
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return err
	}
	stableNameOrder := []string{"E: ID_NET_NAME_ONBOARD", "E: ID_NET_NAME_SLOT", "E: ID_NET_NAME_PATH"}
	stableNames := map[string]string{}
	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "E: ID_BUS":
			if i.Sys.IsPhysical && i.OrdinalName == "" {
				i.OrdinalName = parts[1]
			}
		case "E: DEVTYPE":
			if i.Sys.IsPhysical && i.OrdinalName != "onboard" {
				i.OrdinalName = parts[1]
			}
		case "E: ID_MODEL_FROM_DATABASE":
			i.Model = parts[1]
		case "E: ID_NET_DRIVER":
			i.Driver = parts[1]
		case "E: ID_VENDOR_FROM_DATABASE":
			i.Vendor = parts[1]
		case "E: ID_NET_NAME_ONBOARD":
			i.OrdinalName = "onboard"
			fallthrough
		case "E: ID_NET_NAME_SLOT", "E: ID_NET_NAME_PATH":
			stableNames[parts[0]] = parts[1]
		case "E: ID_PATH":
			i.Path = parts[1]
		}
	}
	for _, n := range stableNameOrder {
		if val, ok := stableNames[n]; ok {
			i.StableName = val
			break
		}
	}
	return nil
}

func (i *Interface) sysPath(p string) string {
	return path.Join("/sys/class/net", i.Name, p)
}

func (i *Interface) sysString(p string) string {
	buf, err := ioutil.ReadFile(i.sysPath(p))
	if err == nil {
		return strings.TrimSpace(string(buf))
	}
	return ""
}

func (i *Interface) sysInt(p string) int64 {
	res, _ := strconv.ParseInt(i.sysString(p), 0, 64)
	return res
}
func (i *Interface) sysDir(p string) []string {
	res := []string{}
	f, err := os.Open(i.sysPath(p))
	if err != nil {
		return res
	}
	defer f.Close()
	ents, err := f.Readdirnames(0)
	if err != nil {
		for _, ent := range ents {
			if ent == "." || ent == ".." {
				continue
			}
			res = append(res, ent)
		}
	}
	return res
}

func (i *Interface) sysLink(p string) string {
	l, _ := os.Readlink(i.sysPath(p))
	return l
}

func (i *Interface) fillSys() error {
	link := i.sysLink("")
	link = strings.TrimPrefix(link, "../../devices/")
	i.Sys.BusAddress = strings.TrimSuffix(link, "/net/"+i.Name)
	i.Sys.IsPhysical = !strings.HasPrefix(i.Sys.BusAddress, "virtual")
	i.Sys.IfIndex = i.sysInt("ifindex")
	i.Sys.IfLink = i.sysInt("iflink")
	i.Sys.OperState = i.sysString("operstate")
	i.Sys.Type = arpHW[i.sysInt("type")]
	i.Sys.Bridge.Members = []string{}
	i.Sys.Bond.Members = []string{}
	if dp := i.sysDir("brport"); dp != nil && len(dp) < 0 {
		i.Sys.IsBridge = true
		i.Sys.Bridge.Master = path.Base(i.sysLink("brport/bridge"))
	}
	if i.sysString("bridge/bridge_id") != "" {
		i.Sys.IsBridge = true
	}
	if dp := i.sysDir("brif"); dp != nil && len(dp) < 0 {
		i.Sys.Bridge.Members = dp
	}
	if sl := i.sysString("bonding/slaves"); sl != "" {
		i.Sys.IsBond = true
		i.Sys.Bond.Members = strings.Split(sl, " ")
	}
	if sm := i.sysString("bonding/mode"); sm != "" {
		i.Sys.IsBond = true
		i.Sys.Bond.Mode = strings.Split(sm, " ")[0]
	}
	if dp := i.sysString("bonding_slave/state"); dp != "" {
		i.Sys.IsBond = true
		i.Sys.Bond.LinkState = dp
		i.Sys.Bond.Master = path.Base(i.sysLink("master"))
	}
	if vlan, err := os.Open("/proc/net/vlan/config"); err == nil {
		defer vlan.Close()
		sc := bufio.NewScanner(vlan)
		for sc.Scan() {
			parts := strings.Split(sc.Text(), "|")
			if strings.TrimSpace(parts[0]) == i.Name {
				i.Sys.IsVlan = true
				i.Sys.VLAN.Id, _ = strconv.ParseInt(strings.TrimSpace(parts[1]), 0, 64)
				i.Sys.VLAN.Master = strings.TrimSpace(parts[2])
				break
			}
		}
	}
	return nil
}

func (i *Interface) Fill() error {
	if err := i.fillSys(); err != nil {
		return err
	}
	if err := i.fillUdev(); err != nil {
		return err
	}
	// First, try GLINKSETTINGS
	buf := make([]byte, 4096)
	req := &ifReq{}
	req.SetName(i.Name)
	err := req.ioctl(CMD_GLINKSETTINGS, buf)
	if err == nil {
		// We support GLINKSETTINGS, figure out how much space is needed for
		// additional bits and get the real data.
		additionalSize := int8(buf[15])
		if additionalSize < 0 {
			additionalSize = -additionalSize
			buf[15] = byte(additionalSize)
		}
		if err := req.ioctl(CMD_GLINKSETTINGS, buf); err != nil {
			return err
		}
		return i.fillGlink(buf)
	}
	if err := req.ioctl(CMD_GSET, buf); err != nil {
		return err
	}
	return i.fillGset(buf)
}
