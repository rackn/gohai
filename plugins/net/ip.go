package net

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"reflect"
	"syscall"
	"unsafe"
)

const (
	SIOCETHTOOL        = 0x8946
	CMD_GSET           = 1
	CMD_GLINKSETTINGS  = 0x4c
	GSET_SIZE          = 44
	GLINKSETTINGS_SIZE = 48
)

var endian binary.ByteOrder

func init() {
	var i int = 0x1
	const INT_SIZE int = int(unsafe.Sizeof(0))
	bs := (*[INT_SIZE]byte)(unsafe.Pointer(&i))
	if bs[0] == 0 {
		endian = binary.BigEndian
	} else {
		endian = binary.LittleEndian
	}
}

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

type ModeBit struct {
	Name, Phy       string
	Feature, Duplex bool
}

func (m ModeBit) String() string {
	if m.Feature {
		return m.Name
	}
	res := fmt.Sprintf("%s base %s", m.Name, m.Phy)
	if m.Duplex {
		return res + " Full"
	}
	return res + " Half"
}

var modeBits = [][8]ModeBit{
	{
		{"10", "T", false, false},
		{"10", "T", false, true},
		{"100", "T", false, false},
		{"100", "T", false, true},
		{"1000", "T", false, false},
		{"1000", "T", false, true},
		{"Autoneg", "", true, false},
		{"TP", "", true, false},
	},
	{
		{"AUI", "", true, false},
		{"MII", "", true, false},
		{"FIBRE", "", true, false},
		{"BNC", "", true, false},
		{"10000", "T", false, true},
		{"Pause", "", true, false},
		{"Asym_Pause", "", true, false},
		{"2500", "X", false, true},
	},
	{
		{"Backplane", "", true, false},
		{"1000", "KX", false, true},
		{"10000", "KX4", false, true},
		{"10000", "KR", false, true},
		{"10000", "R_FEC", false, true},
		{"20000", "MLD2", false, true},
		{"20000", "KR2", false, true},
		{"40000", "KR4", false, true},
	}, {
		{"40000", "CR4", false, true},
		{"40000", "SR4", false, true},
		{"40000", "LR4", false, true},
		{"56000", "KR4", false, true},
		{"56000", "CR4", false, true},
		{"56000", "SR4", false, true},
		{"56000", "LR4", false, true},
		{"25000", "CR", false, true},
	}, {
		{"25000", "KR", false, true},
		{"25000", "SR", false, true},
		{"50000", "CR2", false, true},
		{"50000", "KR2", false, true},
		{"100000", "KR4", false, true},
		{"100000", "SR4", false, true},
		{"100000", "CR4", false, true},
		{"100000", "LR4_ER4", false, true},
	}, {
		{"50000", "SR2", false, true},
		{"1000", "X", false, true},
		{"10000", "CR", false, true},
		{"10000", "SR", false, true},
		{"10000", "LR", false, true},
		{"10000", "LRM", false, true},
		{"10000", "ER", false, true},
		{"2500", "T", false, true},
	}, {
		{"5000", "T", false, true},
	},
}

type Flags net.Flags

func (f Flags) String() string {
	return net.Flags(f).String()
}

func (f Flags) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

type HardwareAddr net.HardwareAddr

func (h HardwareAddr) String() string {
	return net.HardwareAddr(h).String()
}

func (h HardwareAddr) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

type Interface struct {
	Name            string
	MTU             int
	Flags           Flags
	HardwareAddr    HardwareAddr
	Supported       []ModeBit
	Advertised      []ModeBit
	PeerAdvertised  []ModeBit
	Speed           uint32
	Duplex          bool
	Autonegotiation bool
}

func toModeBits(buf []byte) []ModeBit {
	res := []ModeBit{}
	log.Printf("modebuf: %v", buf)
	for segment, bits := range buf {
		if segment >= len(modeBits) {
			break
		}
		for i, modeBit := range modeBits[segment] {
			if bits&(1<<uint(i)) > 0 {
				res = append(res, modeBit)
			}
		}
	}
	return res
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
	log.Printf("buflen: %v, modelen: %v", len(buf), buf[15])
	b := int(buf[15]) << 2
	s := 48
	a := 48 + b
	p := a + b
	log.Printf("s: %d, a: %d, p: %d, e: %d", s, a, p, p+b)
	i.Supported = toModeBits(buf[s : s+b])
	i.Advertised = toModeBits(buf[a : a+b])
	i.PeerAdvertised = toModeBits(buf[p : p+b])
	return nil
}

func (i *Interface) Fill() error {
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

type Info struct {
	Interfaces []Interface
}

func (i *Info) Class() string {
	return "Networking"
}

func Gather() (*Info, error) {
	res := &Info{}
	baseifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	res.Interfaces = make([]Interface, len(baseifs))
	for i, intf := range baseifs {
		iface := Interface{
			Name:           intf.Name,
			HardwareAddr:   HardwareAddr(intf.HardwareAddr),
			MTU:            intf.MTU,
			Flags:          Flags(intf.Flags),
			Supported:      []ModeBit{},
			Advertised:     []ModeBit{},
			PeerAdvertised: []ModeBit{},
		}
		iface.Fill()
		res.Interfaces[i] = iface
	}
	return res, nil
}
