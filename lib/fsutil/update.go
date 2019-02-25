package fsutil

func updateFile(buffer []byte, filename string) (bool, error) {
	if same, err := CompareFile(buffer, filename); err != nil {
		return false, err
	} else if same {
		return false, nil
	} else {
		file, err := CreateRenamingWriter(filename, PublicFilePerms)
		if err != nil {
			return false, err
		}
		defer file.Close()
		if _, err := file.Write(buffer); err != nil {
			return false, err
		}
		return true, nil
	}
}
