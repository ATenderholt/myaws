package utils

type SqlError struct {
	Message string
	Err     error
}

func (e SqlError) Error() string {
	return e.Message + " " + e.Err.Error()
}

type ZipContent struct {
	Offset  int64
	Content []byte
	Length  int64
}

type ZipFileError struct {
	Message  string
	Filepath string
	Err      error
}

func (e ZipFileError) Error() string {
	return e.Message + " " + e.Filepath + " : " + e.Err.Error()
}
