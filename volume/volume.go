package volume

import (
	"strings"
)

// DefaultDriverName is the driver name used for the driver
// implemented in the local package.
const DefaultDriverName string = "local"

// Driver is for creating and removing volumes.
type Driver interface {
	// Name returns the name of the volume driver.
	Name() string
	// Create makes a new volume with the given id.
	Create(name string, opts map[string]string) (Volume, error)
	// Remove deletes the volume.
	Remove(Volume) error
}

// Volume is a place to store data. It is backed by a specific driver, and can be mounted.
type Volume interface {
	// Name returns the name of the volume
	Name() string
	// DriverName returns the name of the driver which owns this volume.
	DriverName() string
	// Path returns the absolute path to the volume.
	Path() string
	// Mount mounts the volume and returns the absolute path to
	// where it can be consumed.
	Mount() (string, error)
	// Unmount unmounts the volume when it is no longer in use.
	Unmount() error
}

// read-write modes
var rwModes = map[string]bool{
	"rw":   true,
	"rw,Z": true,
	"rw,z": true,
	"z,rw": true,
	"Z,rw": true,
	"Z":    true,
	"z":    true,
}

// read-only modes
var roModes = map[string]bool{
	"ro":   true,
	"ro,Z": true,
	"ro,z": true,
	"z,ro": true,
	"Z,ro": true,
}

// mount types
var mountTypes = map[string]bool{
	"nfs": true,
	"ceph": true,
}

// ValidMountMode will make sure the mount mode is valid.
// returns if it's a valid mount mode or not.
func ValidMountMode(mode string) bool {
	return roModes[mode] || rwModes[mode]
}

// ReadWrite tells you if a mode string is a valid read-write mode or not.
func ReadWrite(mode string) bool {
	return rwModes[mode]
}

// ValidMountTypeAndMode checks if the type and mode is valid or not.
// Valid type and mode is a join between a type (ceph, nfs) and a mode.
func ValidMountTypeAndMode(typeAndMode string) bool {
	var types = 0
	var modes = 0
	var z = 0
	for _, item := range strings.Split(typeAndMode, ",") {
		if item == "rw" || item == "ro" {
			modes++
		} else if mountTypes[item] {
			types++
		} else if strings.ToLower(item) == "z" {
			z++
		} else {
			return false
		}
	}
	return modes <= 1 && types <= 1 && z <= 1
}
