package net

import (
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strings"
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

var arpHW = map[int64]string{
	0:      "netrom",
	1:      "ethernet",
	2:      "experimental ethernet",
	3:      "ax25",
	4:      "pronet",
	5:      "chaos",
	6:      "ieee802",
	7:      "arcnet",
	8:      "appletalk",
	15:     "dlci",
	19:     "atm",
	23:     "metricom",
	24:     "ieee1394",
	27:     "eui-64",
	32:     "infiniband",
	256:    "slip",
	257:    "cslip",
	258:    "slip6",
	259:    "cslip6",
	260:    "reserved",
	264:    "adapt",
	270:    "rose",
	271:    "x25",
	272:    "hwx25",
	280:    "can",
	512:    "ppp",
	513:    "hdlc",
	516:    "labp",
	517:    "ddcmp",
	518:    "rawhdlc",
	519:    "rawip",
	768:    "ipip",
	769:    "ip6ip6",
	770:    "frad",
	771:    "skip",
	772:    "loopback",
	773:    "localtalk",
	774:    "fddi",
	775:    "bif",
	776:    "sit",
	777:    "ipddp",
	778:    "ipgre",
	779:    "pimreg",
	780:    "hippi",
	781:    "ash",
	782:    "econet",
	783:    "irda",
	784:    "fcpp",
	785:    "fcal",
	786:    "fcpl",
	787:    "fcfabric",
	800:    "ieee802_tr",
	801:    "ieee80211",
	802:    "ieee80211_prism",
	803:    "ieee80211_radiotap",
	804:    "ieee802154",
	805:    "ieee802154_monitor",
	820:    "phonet",
	821:    "phonet_pipe",
	822:    "caif",
	823:    "ip6gre",
	824:    "netlink",
	825:    "6lowpan",
	826:    "vsockmon",
	0xfffe: "none",
	0xffff: "void",
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

func (f Flags) UnmarshalText(t []byte) error {
	for _, flag := range strings.Split(string(t), "|") {
		switch flag {
		case "0":
		case "up":
			f = f | Flags(net.FlagUp)
		case "broadcast":
			f = f | Flags(net.FlagBroadcast)
		case "loopback":
			f = f | Flags(net.FlagLoopback)
		case "pointtopoint":
			f = f | Flags(net.FlagPointToPoint)
		case "multicast":
			f = f | Flags(net.FlagMulticast)
		default:
			return fmt.Errorf("Unknown flag %s", flag)
		}
	}
	return nil
}

type HardwareAddr net.HardwareAddr

func (h HardwareAddr) String() string {
	return net.HardwareAddr(h).String()
}

func (h HardwareAddr) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

// UnmarshalText unmarshalls the text represenatation of a
// HardwareAddr.  Any format accepted by net.ParseMAC will be
// accepted.
func (h *HardwareAddr) UnmarshalText(buf []byte) error {
	mac, err := net.ParseMAC(string(buf))
	if err != nil {
		return err
	}
	*h = HardwareAddr(mac)
	return nil
}

type IPNet net.IPNet

func (n *IPNet) String() string {
	if len(n.Mask) == 0 {
		return n.IP.String()
	}
	return (*net.IPNet)(n).String()
}

func (n *IPNet) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

// UnmarshalText handles unmarshalling the string represenation of an
// IP address (v4 and v6, in CIDR form and as a raw address) into an
// IP.
func (n *IPNet) UnmarshalText(buf []byte) error {
	addr, cidr, err := net.ParseCIDR(string(buf))
	if err == nil {
		n.IP = addr
		n.Mask = cidr.Mask
		return nil
	}
	n.IP = net.ParseIP(string(buf))
	n.Mask = nil
	return nil
}

// IsCIDR returns whether this IP is in CIDR form.
func (n *IPNet) IsCIDR() bool {
	return len(n.Mask) > 0
}

type Interface struct {
	Name            string
	StableName      string
	OrdinalName     string
	Path            string
	Model           string
	Driver          string
	Vendor          string
	MTU             int
	Flags           Flags
	HardwareAddr    HardwareAddr
	Addrs           []*IPNet
	Supported       []ModeBit
	Advertised      []ModeBit
	PeerAdvertised  []ModeBit
	Speed           uint32
	Duplex          bool
	Autonegotiation bool
	Sys             struct {
		IsPhysical bool
		BusAddress string
		IfIndex    int64
		IfLink     int64
		OperState  string
		Type       string
		IsBridge   bool
		Bridge     struct {
			Members []string
			Master  string
		}
		IsVlan bool
		VLAN   struct {
			Id     int64
			Master string
		}
		IsBond bool
		Bond   struct {
			Mode      string
			Members   []string
			Master    string
			LinkState string
		}
	}
}

func toModeBits(buf []byte) []ModeBit {
	res := []ModeBit{}
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

type Info struct {
	Interfaces    []Interface
	HardwareAddrs map[string]string
	Addrs         map[string]string
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
	res.HardwareAddrs = map[string]string{}
	res.Addrs = map[string]string{}
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
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
			res.HardwareAddrs[iface.HardwareAddr.String()] = iface.Name
		}
		addrs, err := intf.Addrs()
		if err != nil {
			return nil, err
		}
		iface.Addrs = []*IPNet{}
		for i := range addrs {
			addr, ok := addrs[i].(*net.IPNet)
			if ok {
				res.Addrs[addr.String()] = iface.Name
				iface.Addrs = append(iface.Addrs, (*IPNet)(addr))
			}
		}
		iface.Fill()
		res.Interfaces[i] = iface
	}
	sort.SliceStable(res.Interfaces, func(i, j int) bool { return res.Interfaces[i].Path < res.Interfaces[j].Path })
	indexes := map[string]int{}
	for i := range res.Interfaces {
		if idx, ok := indexes[res.Interfaces[i].OrdinalName]; ok {
			indexes[res.Interfaces[i].OrdinalName]++
			res.Interfaces[i].OrdinalName = fmt.Sprintf("%s:%d", res.Interfaces[i].OrdinalName, idx)
		} else if res.Interfaces[i].OrdinalName != "" {
			indexes[res.Interfaces[i].OrdinalName] = 2
			res.Interfaces[i].OrdinalName = fmt.Sprintf("%s:%d", res.Interfaces[i].OrdinalName, 1)
		}
	}
	return res, nil
}
