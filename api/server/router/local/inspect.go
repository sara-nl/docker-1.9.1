package local

import (
	"fmt"
	"net/http"

	"github.com/sara-nl/docker-1.9.1/api/server/httputils"
	"golang.org/x/net/context"
)

// getContainersByName inspects containers configuration and serializes it as json.
func (s *router) getContainersByName(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	displaySize := httputils.BoolValue(r, "size")
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	var json interface{}
	var err error

	version := httputils.VersionFromContext(ctx)

	switch {
	case version.LessThan("1.20"):
		json, err = s.daemon.ContainerInspectPre120(vars["name"])
	case version.Equal("1.20"):
		json, err = s.daemon.ContainerInspect120(vars["name"])
	default:
		json, err = s.daemon.ContainerInspect(vars["name"], displaySize)
	}

	if err != nil {
		return err
	}

	return httputils.WriteJSON(w, http.StatusOK, json)
}
