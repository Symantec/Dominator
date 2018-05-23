package topology

func (directory *Directory) walk(fn func(*Directory) error) error {
	if err := fn(directory); err != nil {
		return err
	}
	for _, subdir := range directory.Directories {
		if err := subdir.walk(fn); err != nil {
			return err
		}
	}
	return nil
}
