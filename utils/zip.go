package utils

import (
	"archive/zip"
	"errors"
	"io"
	"myaws/log"
	"os"
	"path/filepath"
)

func min(a int, b int64) int64 {
	a64 := int64(a)
	if a64 < b {
		return a64
	}

	return b
}

func (source ZipContent) ReadAt(p []byte, off int64) (n int, err error) {
	log.Debug("Attempting to read %d bytes from offset %d", len(p), off)

	if off >= source.Length {
		return 0, io.EOF
	}

	bytesToRead := min(len(p), source.Length-off)
	count := copy(p, source.Content[off:off+bytesToRead])

	if count < len(p) {
		return count, io.EOF
	}

	return count, nil
}

func saveFile(filePath string, file zip.File) error {
	destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		msg := log.Error("unable to open file %s: %v", filePath, err)
		return errors.New(msg)
	}
	defer destFile.Close()

	fileInArchive, err := file.Open()
	if err != nil {
		msg := log.Error("unable to decompress file %s: %v", file.Name, err)
		return errors.New(msg)
	}
	defer fileInArchive.Close()

	_, err = io.Copy(destFile, fileInArchive)
	if err != nil {
		msg := log.Error("problem decompressing file %s: %v", file.Name, err)
		return errors.New(msg)
	}

	return nil
}

func UncompressZipFile(file string, destPath string) error {
	reader, err := zip.OpenReader(file)
	if err != nil {
		msg := log.Error("unable to uncompress zip from file %s: %v", file, err)
		return errors.New(msg)
	}
	defer reader.Close()

	return decompressZipFile(&reader.Reader, destPath)
}

func UncompressZipFileBytes(bytes []byte, destPath string) error {
	content := ZipContent{Content: bytes, Length: int64(len(bytes))}
	reader, err := zip.NewReader(content, content.Length)
	if err != nil {
		msg := log.Error("Unable to uncompress zip from bytes: %v", err)
		return errors.New(msg)
	}

	return decompressZipFile(reader, destPath)
}

func decompressZipFile(reader *zip.Reader, destPath string) error {
	for _, f := range reader.File {
		filePath := filepath.Join(destPath, f.Name)

		var err error
		if f.FileInfo().IsDir() {
			err = CreateDirs(filePath)
			continue
		} else {
			err = CreateDirs(filepath.Dir(filePath))
		}

		if err != nil {
			msg := log.Error("unable to create directory for file %s in %s: %v", f.Name, destPath, err)
			return errors.New(msg)
		}

		log.Info("Saving %s ...", filePath)
		err = saveFile(filePath, *f)
		if err != nil {
			msg := log.Error("unable to save file %s in %s: %v", f.Name, destPath, err)
			return errors.New(msg)
		}

	}

	return nil
}
