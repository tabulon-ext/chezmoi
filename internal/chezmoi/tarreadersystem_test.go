package chezmoi

import (
	"archive/tar"
	"bytes"
	"errors"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTARReaderSystem(t *testing.T) {
	b := &bytes.Buffer{}
	w := tar.NewWriter(b)
	assert.NoError(t, w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     "dir/",
		Mode:     0o777,
	}))
	data := []byte("# contents of dir/file\n")
	assert.NoError(t, w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "dir/file",
		Size:     int64(len(data)),
		Mode:     0o666,
	}))
	_, err := w.Write(data)
	assert.NoError(t, err)
	linkname := "file"
	assert.NoError(t, w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     "dir/symlink",
		Linkname: linkname,
	}))
	require.NoError(t, w.Close())

	tarReaderSystem, err := NewTARReaderSystem(tar.NewReader(b), TARReaderSystemOptions{
		RootAbsPath:     "/home/user",
		StripComponents: 1,
	})
	assert.NoError(t, err)

	for _, tc := range []struct {
		absPath      AbsPath
		lstatErr     error
		readlink     string
		readlinkErr  error
		readFileData []byte
		readFileErr  error
	}{
		{
			absPath:      "/home/user/file",
			readlinkErr:  fs.ErrInvalid,
			readFileData: data,
		},
		{
			absPath:     "/home/user/notexist",
			readlinkErr: fs.ErrNotExist,
			lstatErr:    fs.ErrNotExist,
			readFileErr: fs.ErrNotExist,
		},
		{
			absPath:     "/home/user/symlink",
			readlink:    "file",
			readFileErr: fs.ErrInvalid,
		},
	} {
		_, err = tarReaderSystem.Lstat(tc.absPath)
		if tc.lstatErr != nil {
			assert.True(t, errors.Is(err, tc.lstatErr))
		} else {
			assert.NoError(t, err)
		}

		actualLinkname, err := tarReaderSystem.Readlink(tc.absPath)
		if tc.readlinkErr != nil {
			assert.True(t, errors.Is(err, tc.readlinkErr))
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.readlink, actualLinkname)
		}

		actualReadFileData, err := tarReaderSystem.ReadFile(tc.absPath)
		if tc.readFileErr != nil {
			assert.True(t, errors.Is(err, tc.readFileErr))
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.readFileData, actualReadFileData)
		}
	}
}
