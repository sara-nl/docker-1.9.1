package builder

import (
	"os"

	"github.com/sara-nl/docker-1.9.1/pkg/archive"
	"github.com/sara-nl/docker-1.9.1/utils"
)

// MakeGitContext returns a Context from gitURL that is cloned in a temporary directory.
func MakeGitContext(gitURL string) (ModifiableContext, error) {
	root, err := utils.GitClone(gitURL)
	if err != nil {
		return nil, err
	}

	c, err := archive.Tar(root, archive.Uncompressed)
	if err != nil {
		return nil, err
	}

	defer func() {
		// TODO: print errors?
		c.Close()
		os.RemoveAll(root)
	}()
	return MakeTarSumContext(c)
}
