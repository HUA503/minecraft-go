package world

// ReplaceFile atomically replaces dst with src using the platform's durable
// same-volume replacement operation. Both paths should be on the same volume.
func ReplaceFile(src, dst string) error {
	return replaceFile(src, dst)
}
