package compressor

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func WriteTar(srcPath string, dest io.Writer) error {
	return WriteTarExclude(srcPath, dest, nil)
}

func WriteTarExclude(srcPath string, dest io.Writer, excludeList []string) error {
	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	tw := tar.NewWriter(dest)
	defer tw.Close()

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var relative string
		if os.IsPathSeparator(srcPath[len(srcPath)-1]) {
			relative, err = filepath.Rel(absPath, path)
		} else {
			relative, err = filepath.Rel(filepath.Dir(absPath), path)
		}

		relative = filepath.ToSlash(relative)

		if err != nil {
			return err
		}

		if fileIsExcluded(path, excludeList) {
			return nil
		}

		return addTarFile(path, relative, tw)
	})

	return err
}

func addTarFile(path, name string, tw *tar.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	link := ""
	if fi.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(path); err != nil {
			return err
		}
	}

	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return err
	}

	if fi.IsDir() && !os.IsPathSeparator(name[len(name)-1]) {
		name = name + "/"
	}

	if hdr.Typeflag == tar.TypeReg && name == "." {
		// archiving a single file
		hdr.Name = filepath.ToSlash(filepath.Base(path))
	} else {
		hdr.Name = filepath.ToSlash(name)
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if hdr.Typeflag == tar.TypeReg {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		defer file.Close()

		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func fileIsExcluded(path string, excludeList []string) bool {
	for _, f := range excludeList {
		if strings.HasPrefix(path, f) {
			return true
		}
	}
	return false
}
