package topology

func walk(directory *Directory, fn func(*Directory) error) error {
	if err := fn(directory); err != nil {
		return err
	}
	for _, subdir := range directory.Directories {
		if err := walk(subdir, fn); err != nil {
			return err
		}
	}
	return nil
}
