package cephvolumedriver

import (
	"errors"
	"sync"
	"runtime/debug"
	"bytes"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/volume"
	"os/exec"
	//"strconv"
	"strings"
)

const CephImageSizeMB = 1024 * 1024 // 1TB

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
	fmt.Printf("=== Root.Create('%s') ===\n", name)
	debug.PrintStack()
	r.m.Lock()
	defer r.m.Unlock()

	v, exists := r.volumes[name]
	if !exists {
		//TODO: Might want to map with --options rw/ro here, but then we need to sneak in the RW flag somehow
		//var stdout bytes.Buffer
		//var stderr bytes.Buffer
		var mappedDevicePath string

		/*cmd := exec.Command("echo", "rbd", "create", name, "--size", strconv.Itoa(CephImageSizeMB))
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			logrus.Infof("Created Ceph volume %s", name)
			mappedDevicePath, err = mapCephVolume(name)
			if err != nil {
				return nil, err
			}
			cmd = exec.Command("echo", "mkfs.ext4", "-m0", mappedDevicePath)
			logrus.Infof("Creating ext4 filesystem in newly created Ceph volume %s (device %s)", name, mappedDevicePath)
			if err := cmd.Run(); err != nil {
				logrus.Errorf("Failed to create ext4 filesystem in newly created Ceph volume %s (device %s) - %s", name, mappedDevicePath, err)
				return nil, err
			}
		} else if strings.Contains(stderr.String(), fmt.Sprintf("rbd image %s already exists", name)) {*/
			logrus.Infof("Found existing Ceph volume %s", name)
			mappedDevicePath, err := mapCephVolume(name)
			if err != nil {
				return nil, err
			}
		/*} else {
			msg := fmt.Sprintf("Failed to create Ceph volume %s - %s", name, err)
			logrus.Errorf(msg)
			return nil, errors.New(msg)
		}*/

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

func mapCephVolume(name string) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("echo", "rbd", "map", name)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	var mappedDevicePath string
	if err := cmd.Run(); err == nil {
		mappedDevicePath = "/dev/loop0"// strings.TrimRight(stdout.String(), "\n")
		logrus.Infof("Succeeded in mapping Ceph volume %s to %s", name, mappedDevicePath)
		return mappedDevicePath, nil
	} else {
		msg := fmt.Sprintf("Failed to map Ceph volume %s: %s - %s", name, err, strings.TrimRight(stderr.String(), "\n"))
		logrus.Errorf(msg)
		return "", errors.New(msg)
	}

}

func (r *Root) Remove(v volume.Volume) error {
	fmt.Printf("=== Root.Remove('%s', '%s') ===\n", v.Name(), v.Path())
	debug.PrintStack()
	r.m.Lock()
	defer r.m.Unlock()

	lv, ok := v.(*Volume)
	if !ok {
		return errors.New("unknown volume type")
	}
	fmt.Printf("=== UsedCount for %s: %d ===\n", v.Name(), lv.usedCount)
	lv.release()
	if lv.usedCount == 0 {
		unmapCephVolume(lv.name, lv.mappedDevicePath)
		delete(r.volumes, lv.name)
	}
	return nil
}

func unmapCephVolume(name, mappedDevicePath string) error {
	cmd := exec.Command("echo", "rbd", "unmap", mappedDevicePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		logrus.Infof("Succeeded in unmapping Ceph volume %s from %s", name, mappedDevicePath)
	} else {
		logrus.Printf("Failed to unmap Ceph volume %s from %s: %s - %s", name, mappedDevicePath, err, strings.TrimRight(stderr.String(), "\n"))
	}
	return err
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
	fmt.Printf("=== Volume.Mount('%s') ===\n", v.Name())
	debug.PrintStack()
	// The return value from this method will be passed to the container
	return v.mappedDevicePath, nil
}

func (v *Volume) Unmount() error {
	fmt.Printf("=== Volume.Unmount('%s') ===\n", v.Name())
	debug.PrintStack()
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
