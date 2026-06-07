package platform

// FileID — кросс-платформенная идентичность файла для детекта ротации/перезаписи.
// На Unix — dev+inode; на Windows — volume serial + file index.
type FileID struct {
	A uint64 // Unix: st_dev; Windows: VolumeSerialNumber
	B uint64 // Unix: st_ino; Windows: FileIndexHigh<<32|FileIndexLow
}

func (f FileID) IsZero() bool { return f.A == 0 && f.B == 0 }
