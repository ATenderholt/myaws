package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
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
	log.Printf("Attempting to read %d bytes from offset %d", len(p), off)

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

func createDirs(dirPath string) {
	log.Printf("Creating directory if necessary %s ...", dirPath)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		panic(ZipFileError{"unable to create directory", dirPath, err})
	}
}

func saveFile(filePath string, file zip.File) {
	destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		panic(ZipFileError{"unable to create file", filePath, err})
	}
	defer destFile.Close()

	fileInArchive, err := file.Open()
	if err != nil {
		panic(ZipFileError{"unable to open decompressed file", file.Name, err})
	}
	defer fileInArchive.Close()

	_, err = io.Copy(destFile, fileInArchive)
	if err != nil {
		panic(ZipFileError{"problem decompressing file", filePath, err})
	}
}

func DecompressZipFile(bytes []byte, destPath string) (returnError error) {
	content := ZipContent{Content: bytes, Length: int64(len(bytes))}
	reader, err := zip.NewReader(content, content.Length)

	if err != nil {
		return fmt.Errorf("error when reading zip: %v", err)
	}

	defer func() {
		if e := recover(); e != nil {
			// cleanup?
			err := e.(ZipFileError)
			log.Printf(err.Error())
			returnError = err
		}
	}()

	for _, f := range reader.File {
		filePath := filepath.Join(destPath, f.Name)

		if f.FileInfo().IsDir() {
			createDirs(filePath)
			continue
		} else {
			createDirs(filepath.Dir(filePath))
		}

		log.Printf("Saving %s ...", filePath)
		saveFile(filePath, *f)

	}

	return nil
}
