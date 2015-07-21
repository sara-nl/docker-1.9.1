package cephvolumedriver

import (
	"errors"
	"sync"
	"runtime/debug"

	"github.com/docker/docker/volume"
)

func New() *Root {
	debug.PrintStack()
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

func (r *Root) Create(name string) (volume.Volume, error) {
	debug.PrintStack()
	r.m.Lock()
	defer r.m.Unlock()

	v, exists := r.volumes[name]
	if !exists {
		//TODO: Ceph map
		v = &Volume{
			driverName: r.Name(),
			name:       name,
		}
		r.volumes[name] = v
	}
	v.use()
	return v, nil
}

func (r *Root) Remove(v volume.Volume) error {
	debug.PrintStack()
	r.m.Lock()
	defer r.m.Unlock()

	lv, ok := v.(*Volume)
	if !ok {
		return errors.New("unknown volume type")
	}
	lv.release()
	if lv.usedCount == 0 {
		//TODO: Ceph unmap
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
}

func (v *Volume) Name() string {
	debug.PrintStack()
	return v.name
}

func (v *Volume) DriverName() string {
	debug.PrintStack()
	return v.driverName
}

func (v *Volume) Path() string {
	debug.PrintStack()
	return ""
}

func (v *Volume) Mount() (string, error) {
	debug.PrintStack()
	return "", nil
}

func (v *Volume) Unmount() error {
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
