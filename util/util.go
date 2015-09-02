package util

import "os"

//	Return true when specified path is exist
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
