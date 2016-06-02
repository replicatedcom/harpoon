package compressor

import (
	"compress/gzip"
	"os"
)

type Compressor interface {
	Compress(src string, dst string) error
	CompressExclude(src string, dest string, excludeList []string) error
}

func NewTgz() Compressor {
	return &tgzCompressor{}
}

type tgzCompressor struct{}

func (compressor *tgzCompressor) Compress(src string, dest string) error {
	return compressor.CompressExclude(src, dest, nil)
}

func (compressor *tgzCompressor) CompressExclude(src string, dest string, excludeList []string) error {
	fw, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fw.Close()

	gw := gzip.NewWriter(fw)
	defer gw.Close()

	return WriteTarExclude(src, gw, excludeList)
}
