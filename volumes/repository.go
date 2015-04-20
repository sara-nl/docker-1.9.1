package volumes

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/utils"
)

type Repository struct {
	configPath string
	driver     graphdriver.Driver
	volumes    map[string]*Volume
	lock       sync.Mutex
}

func NewRepository(configPath string, driver graphdriver.Driver) (*Repository, error) {
	abspath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	// Create the config path
	if err := os.MkdirAll(abspath, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}

	repo := &Repository{
		driver:     driver,
		configPath: abspath,
		volumes:    make(map[string]*Volume),
	}

	return repo, repo.restore()
}

func (r *Repository) newVolume(path string, writable bool, ceph bool, isStarting bool) (*Volume, error) {
	var (
		isBindMount bool
		err         error
		id          = utils.GenerateRandomID()
	)

	cephVolume := ""
	cephDevice := ""
	if path == "" {
		path, err = r.createNewVolumePath(id)
		if err != nil {
			return nil, err
		}
	} else if ceph {
		isBindMount = true
		cephVolume = path
		cephDevice = "/dev/rbd/rbd/" + cephVolume //TODO: Allow for this to be overridden by an environment variable? TODO: This might not actually be the correct path, if the volume is mapped multiple times. How to get the device number (e.g. /dev/rbd0)?
		path = "/var/lib/docker/cephmount-" + utils.GenerateRandomID() + "-" + cephVolume //TODO: Use m.container.basefs?
		//TODO: Might want to check the directory for existence and retry if it does exist and is nonempty (or already has something mounted to it)
		fmt.Printf("Ceph: %s -> %s -> %s\n", cephVolume, cephDevice, path)
	} else {
		isBindMount = true
		path = filepath.Clean(path)
	}

	// Ignore the error here since the path may not exist
	// Really just want to make sure the path we are using is real(or non-existant)
	if cleanPath, err := filepath.EvalSymlinks(path); err == nil {
		path = cleanPath
	}

	v := &Volume{
		ID:          id,
		Path:        path,
		repository:  r,
		Writable:    writable,
		CephVolume:  cephVolume,
		CephDevice:  cephDevice,
		containers:  make(map[string]struct{}),
		configPath:  r.configPath + "/" + id,
		IsBindMount: isBindMount,
	}

	if err := v.initialize(isStarting); err != nil {
		return nil, err
	}

	return v, r.add(v)
}

func (r *Repository) restore() error {
	dir, err := ioutil.ReadDir(r.configPath)
	if err != nil {
		return err
	}

	for _, v := range dir {
		id := v.Name()
		vol := &Volume{
			ID:         id,
			configPath: r.configPath + "/" + id,
			containers: make(map[string]struct{}),
		}
		if err := vol.FromDisk(); err != nil {
			if !os.IsNotExist(err) {
				log.Debugf("Error restoring volume: %v", err)
				continue
			}
			if err := vol.initialize(false); err != nil {
				log.Debugf("%s", err)
				continue
			}
		}
		if err := r.add(vol); err != nil {
			log.Debugf("Error restoring volume: %v", err)
		}
	}
	return nil
}

func (r *Repository) Get(path string) *Volume {
	r.lock.Lock()
	vol := r.get(path)
	r.lock.Unlock()
	return vol
}

func (r *Repository) get(path string) *Volume {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil
	}
	return r.volumes[filepath.Clean(path)]
}

func (r *Repository) add(volume *Volume) error {
	if vol := r.get(volume.Path); vol != nil {
		return fmt.Errorf("Volume exists: %s", volume.ID)
	}
	r.volumes[volume.Path] = volume
	return nil
}

func (r *Repository) Delete(path string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	volume := r.get(filepath.Clean(path))
	if volume == nil {
		return fmt.Errorf("Volume %s does not exist", path)
	}

	containers := volume.Containers()
	if len(containers) > 0 {
		return fmt.Errorf("Volume %s is being used and cannot be removed: used by containers %s", volume.Path, containers)
	}

	if err := os.RemoveAll(volume.configPath); err != nil {
		return err
	}

	if !volume.IsBindMount {
		if err := r.driver.Remove(volume.ID); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	delete(r.volumes, volume.Path)
	return nil
}

func (r *Repository) createNewVolumePath(id string) (string, error) {
	if err := r.driver.Create(id, ""); err != nil {
		return "", err
	}

	path, err := r.driver.Get(id, "")
	if err != nil {
		return "", fmt.Errorf("Driver %s failed to get volume rootfs %s: %v", r.driver, id, err)
	}

	return path, nil
}

func (r *Repository) FindOrCreateVolume(path string, writable bool, ceph bool, isStarting bool) (*Volume, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if path == "" {
		return r.newVolume(path, writable, ceph, isStarting)
	}

	if v := r.get(path); v != nil {
		return v, nil
	}

	return r.newVolume(path, writable, ceph, isStarting)
}
