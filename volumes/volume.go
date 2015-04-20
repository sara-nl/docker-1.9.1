package volumes

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"os/exec"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/symlink"
	"fmt"
	"bytes"
)

type Volume struct {
	ID          string
	Path        string
	CephVolume  string
	CephDevice  string
	IsBindMount bool
	Writable    bool
	containers  map[string]struct{}
	configPath  string
	repository  *Repository
	lock        sync.Mutex
}

func (v *Volume) Export(resource, name string) (io.ReadCloser, error) {
	if v.IsBindMount && filepath.Base(resource) == name {
		name = ""
	}

	basePath, err := v.getResourcePath(resource)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(basePath)
	if err != nil {
		return nil, err
	}
	var filter []string
	if !stat.IsDir() {
		d, f := path.Split(basePath)
		basePath = d
		filter = []string{f}
	} else {
		filter = []string{path.Base(basePath)}
		basePath = path.Dir(basePath)
	}
	return archive.TarWithOptions(basePath, &archive.TarOptions{
		Compression:  archive.Uncompressed,
		Name:         name,
		IncludeFiles: filter,
	})
}

func (v *Volume) IsDir() (bool, error) {
	stat, err := os.Stat(v.Path)
	if err != nil {
		return false, err
	}

	return stat.IsDir(), nil
}

func (v *Volume) Containers() []string {
	v.lock.Lock()

	var containers []string
	for c := range v.containers {
		containers = append(containers, c)
	}

	v.lock.Unlock()
	return containers
}

func (v *Volume) RemoveContainer(containerId string) {
	v.lock.Lock()
	delete(v.containers, containerId)
	v.lock.Unlock()
}

func (v *Volume) AddContainer(containerId string) {
	v.lock.Lock()
	v.containers[containerId] = struct{}{}
	v.lock.Unlock()
}

func (v *Volume) initialize(isStarting bool) error {
	v.lock.Lock()
	defer v.lock.Unlock()
	fmt.Printf("Initializing volume: %s %s %s\n", v.Path, v.CephVolume, v.CephDevice)

	if _, err := os.Stat(v.Path); err != nil && os.IsNotExist(err) {
		fmt.Printf("Creating %s on host\n", v.Path)
		if err := os.MkdirAll(v.Path, 0755); err != nil {
			return err
		}
	}

	if (v.CephVolume != "" && isStarting) {
		fmt.Printf("Mapping %s\n", v.CephVolume)
		err := exec.Command("rbd", "map", v.CephVolume).Run()
		if err == nil {
			fmt.Printf("Succeeded executing rbd\n")
		} else {
			fmt.Printf("Error executing rbd: %s\n", err)
		}
		fmt.Printf("Mounting %s to %s on host\n", v.CephDevice, v.Path)
		var out bytes.Buffer
		cmd := exec.Command("mount", v.CephDevice, v.Path)
		cmd.Stderr = &out
		err = cmd.Run()
		if err == nil {
			fmt.Printf("Succeeded mounting\n")
		} else {
			fmt.Printf("Error mounting: %s - %s\n", err, out.String())
		}
	}

	if err := os.MkdirAll(v.configPath, 0755); err != nil {
		return err
	}

	jsonPath, err := v.jsonPath()
	if err != nil {
		return err
	}
	f, err := os.Create(jsonPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return v.toDisk()
}

func (v *Volume) ToDisk() error {
	v.lock.Lock()
	defer v.lock.Unlock()
	return v.toDisk()
}

func (v *Volume) toDisk() error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	pth, err := v.jsonPath()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pth, data, 0666)
}

func (v *Volume) FromDisk() error {
	v.lock.Lock()
	defer v.lock.Unlock()
	pth, err := v.jsonPath()
	if err != nil {
		return err
	}

	jsonSource, err := os.Open(pth)
	if err != nil {
		return err
	}
	defer jsonSource.Close()

	dec := json.NewDecoder(jsonSource)

	return dec.Decode(v)
}

func (v *Volume) jsonPath() (string, error) {
	return v.getRootResourcePath("config.json")
}
func (v *Volume) getRootResourcePath(path string) (string, error) {
	cleanPath := filepath.Join("/", path)
	return symlink.FollowSymlinkInScope(filepath.Join(v.configPath, cleanPath), v.configPath)
}

func (v *Volume) getResourcePath(path string) (string, error) {
	cleanPath := filepath.Join("/", path)
	return symlink.FollowSymlinkInScope(filepath.Join(v.Path, cleanPath), v.Path)
}
