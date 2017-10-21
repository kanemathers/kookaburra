package main

import (
	"io"

	"github.com/anacrolix/torrent"
)

type SeekableFile struct {
	*torrent.File
	*torrent.Reader
}

func (self *SeekableFile) Seek(off int64, whence int) (ret int64, err error) {
	var pos int64

	switch whence {
	case io.SeekStart:
		pos = self.File.Offset() + off
	case io.SeekCurrent:
		pos = off
	case io.SeekEnd:
		pos = (self.File.Offset() + self.File.Length()) - off
	}

	return self.Reader.Seek(pos, whence)
}

type Torrent struct {
	*torrent.Torrent
}

func (self *Torrent) Files() []*SeekableFile {
	files := self.Torrent.Files()
	seekableFiles := make([]*SeekableFile, len(files))

	for i := 0; i < len(files); i++ {
		seekableFile := &SeekableFile{
			File:   &files[i],
			Reader: self.Torrent.NewReader(),
		}

		seekableFile.Reader.SetResponsive()
		seekableFile.Reader.SetReadahead(5 * 1024 * 1024)

		seekableFiles[i] = seekableFile
	}

	return seekableFiles
}

func (self *Torrent) LargestFile() *SeekableFile {
	files := self.Files()
	largest := files[0]

	for _, file := range files[1:] {
		if file.Length() > largest.Length() {
			largest = file
		}
	}

	return largest
}
