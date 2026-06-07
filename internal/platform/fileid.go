package platform

// FileID — кросс-платформенная идентичность файла для детекта ротации/перезаписи.
// На Unix — dev+inode; на Windows — volume serial + file index.
type FileID struct {
	A uint64
	B uint64
}

func (f FileID) IsZero() bool { return f.A == 0 && f.B == 0 }
