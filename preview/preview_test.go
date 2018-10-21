package preview

import (
	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var snapshotter = cupaloy.New(cupaloy.SnapshotFileExtension(".html"))

func TestSnapshotTestData(t *testing.T) {
	files, err := ioutil.ReadDir("testdata")
	require.NoError(t, err)

	for _, f := range files {
		t.Run(f.Name(), func(t *testing.T) {
			file, err := os.Open(filepath.Join("testdata", f.Name()))
			require.NoError(t, err)

			fileBytes, err := ioutil.ReadAll(file)
			require.NoError(t, err)

			page, err := GetPageForFile(string(fileBytes))
			require.NoError(t, err)
			snapshotter.SnapshotT(t, page)
		})
	}
}
