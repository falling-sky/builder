package postprocess

import "log"

// HTML will do whatever is needed for CSS files specifically.
func HTML(filename string, content string) error {
	log.Printf("Need to write: %s (%v bytes)\n", filename, len(content))

	return nil
}
