package main

import (
	"os"
	"time"
)

type fakeFile string

func (f fakeFile) Name() string {
	return string(f)
}

func (f fakeFile) Size() int64 {
	return int64(len(f))
}

func (f fakeFile) Mode() os.FileMode {
	return 0777
}

func (f fakeFile) ModTime() time.Time {
	return time.Now()
}

func (f fakeFile) IsDir() bool {
	return false
}

func (f fakeFile) Sys() interface{} {
	return nil
}
