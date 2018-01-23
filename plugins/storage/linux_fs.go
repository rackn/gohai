package storage

import (
	"bufio"
	"os"
	"strings"
	"syscall"
)

// At some point, also need to add mode block device oriented information here

type Volume struct {
	BackingDevice string
	Filesystem    string
	Name          string
	Options       string
	Virtual       bool
	Blocks        struct {
		Size  int64
		Total uint64
		Free  uint64
		Avail uint64
	}
}

type Info struct {
	Volumes []Volume
}

func (i *Info) Class() string {
	return "Storage"
}

func Gather() (*Info, error) {
	res := &Info{
		Volumes: []Volume{},
	}

	mounts, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil, err
	}
	defer mounts.Close()
	mountLines := bufio.NewScanner(mounts)
	for mountLines.Scan() {
		line := mountLines.Text()
		fields := strings.Split(line, " ")
		if len(fields) != 6 {
			continue
		}
		vol := Volume{
			Name:          fields[1],
			BackingDevice: fields[0],
			Filesystem:    fields[2],
			Options:       fields[3],
		}
		stat, err := os.Stat(vol.BackingDevice)
		if err == nil {
			vol.Virtual = !(stat.Mode()&os.ModeDevice > 0)
		}
		fsStat := &syscall.Statfs_t{}
		if err := syscall.Statfs(vol.Name, fsStat); err == nil {
			vol.Blocks.Size = int64(fsStat.Bsize)
			vol.Blocks.Total = fsStat.Blocks
			vol.Blocks.Free = fsStat.Bfree
			vol.Blocks.Avail = fsStat.Bavail
		}
		res.Volumes = append(res.Volumes, vol)
	}
	return res, nil
}
