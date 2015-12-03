package cephvolumedriver

import (
	"errors"
	"sync"

	"github.com/docker/docker/volume"
	"github.com/Sirupsen/logrus"
	"os/exec"
	"bytes"
	"strings"
	"fmt"
)

func New() *Root {
	return &Root{
		volumes: make(map[string]*Volume),
	}
}

type Root struct {
	m       sync.Mutex
	volumes map[string]*Volume
}

func (r *Root) Name() string {
	return "ceph"
}

func (r *Root) Create(name string, _ map[string]string) (volume.Volume, error) {
	r.m.Lock()
	defer r.m.Unlock()

	v, exists := r.volumes[name]
	if !exists {
		//TODO: Might want to map with --options rw/ro here, but then we need to sneak in the RW flag somehow
		cmd := exec.Command("rbd", "map", name)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		var mappedDevicePath string
		if err := cmd.Run(); err == nil {
			mappedDevicePath = strings.TrimRight(stdout.String(), "\n")
			logrus.Infof("Succeeded in mapping Ceph volume %s to %s", name, mappedDevicePath)
		} else {
			msg := fmt.Sprintf("Failed to map Ceph volume %s: %s - %s", name, err, strings.TrimRight(stderr.String(), "\n"))
			logrus.Errorf(msg)
			return nil, errors.New(msg)
		}
		v = &Volume{
			driverName:       r.Name(),
			name:             name,
			mappedDevicePath: mappedDevicePath,
		}
		r.volumes[name] = v
	}
	v.use()
	return v, nil
}

func (r *Root) Remove(v volume.Volume) error {
	r.m.Lock()
	defer r.m.Unlock()

	lv, ok := v.(*Volume)
	if !ok {
		return errors.New("unknown volume type")
	}
	lv.release()
	if lv.usedCount == 0 {
		cmd := exec.Command("rbd", "unmap", lv.mappedDevicePath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			logrus.Infof("Succeeded in unmapping Ceph volume %s from %s", lv.name, lv.mappedDevicePath)
		} else {
			logrus.Errorf("Failed to unmap Ceph volume %s from %s: %s - %s", lv.name, lv.mappedDevicePath, err, strings.TrimRight(stderr.String(), "\n"))
		}
		delete(r.volumes, lv.name)
	}
	return nil
}

type Volume struct {
	m         sync.Mutex
	usedCount int
	// unique name of the volume
	name string
	// driverName is the name of the driver that created the volume.
	driverName string
	// the path to the device to which the Ceph volume has been mapped
	mappedDevicePath string
}

func (v *Volume) Name() string {
	return v.name
}

func (v *Volume) DriverName() string {
	return v.driverName
}

func (v *Volume) Path() string {
	return ""
}

func (v *Volume) Mount() (string, error) {
	// The return value from this method will be passed to the container
	return v.mappedDevicePath, nil
}

func (v *Volume) Unmount() error {
	return nil
}

func (v *Volume) use() {
	v.m.Lock()
	v.usedCount++
	v.m.Unlock()
}

func (v *Volume) release() {
	v.m.Lock()
	v.usedCount--
	v.m.Unlock()
}
