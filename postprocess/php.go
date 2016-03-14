package postprocess

import "log"

// PHP will do whatever is needed for CSS files specifically.
func PHP(filename string, content string) error {
	log.Printf("Need to write: %s (%v bytes)\n", filename, len(content))
	return nil
}
