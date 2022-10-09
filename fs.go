package main

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"time"
)

var (
	_ fs.File       = &PluginFile{}
	_ io.ReadCloser = &PluginFile{}
	_ io.Writer     = &PluginFile{}
	_ fs.FileInfo   = &PluginFile{}
	_ fs.FS         = &PluginFS{}
)

type PluginFile struct {
	name   string
	reader io.Reader
	writer io.WriteCloser
	mode   bool
}

func (f *PluginFile) Name() string               { return f.name }
func (f *PluginFile) Size() int64                { return 0 }
func (f *PluginFile) ModTime() time.Time         { return time.Time{} }
func (f *PluginFile) IsDir() bool                { return false }
func (f *PluginFile) Sys() any                   { return nil }
func (f *PluginFile) Type() fs.FileMode          { return f.Mode().Type() }
func (f *PluginFile) Info() (fs.FileInfo, error) { return f, nil }
func (f *PluginFile) Mode() fs.FileMode          { return 0444 }

// Close implements fs.File
func (i *PluginFile) Close() error {
	fmt.Println("Close called")
	if i.mode {
		fmt.Println("Writer close called")
		i.writer.Close()
	}
	return nil
}

// Stat implements fs.File
func (i *PluginFile) Stat() (fs.FileInfo, error) {
	return i, nil
}

// Read implements io.Reader
func (i *PluginFile) Read(p []byte) (n int, err error) {
	return i.reader.Read(p)
}

// Write implements io.Writeer
func (i *PluginFile) Write(p []byte) (n int, err error) {
	return i.writer.Write(p)
}

type PluginFS struct {
	fsMu     sync.Mutex
	inFiles  map[string]*PluginFile
	outFiles map[string]*PluginFile
}

// Open implements fs.FS
func (s *PluginFS) Open(name string) (fs.File, error) {
	s.fsMu.Lock()
	defer s.fsMu.Unlock()
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return nil, fs.ErrPermission
	}

	switch parts[0] {
	case "in":
		f, ok := s.inFiles[parts[1]]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return f, nil
	case "out":
		f, ok := s.outFiles[parts[1]]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return f, nil
	default:
		return nil, fs.ErrPermission
	}
}

func (s *PluginFS) Register(id string, inFile, outFile *PluginFile) error {
	s.fsMu.Lock()
	defer s.fsMu.Unlock()
	if _, ok := s.inFiles[id]; ok {
		return fs.ErrExist
	}
	if _, ok := s.outFiles[id]; ok {
		return fs.ErrExist
	}
	s.inFiles[id] = inFile
	s.outFiles[id] = outFile
	return nil
}
