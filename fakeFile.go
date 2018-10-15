package main

import (
	"os"
	"time"
)

type FakeFile string

func (f FakeFile) Name() string {
	return string(f)
}

func (f FakeFile) Size() int64 {
	return int64(len(f))
}

func (f FakeFile) Mode() os.FileMode {
	return 0777
}

func (f FakeFile) ModTime() time.Time {
	return time.Now()
}

func (f FakeFile) IsDir() bool {
	return false
}

func (f FakeFile) Sys() interface{} {
	return nil
}
