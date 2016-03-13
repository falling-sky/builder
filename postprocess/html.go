package postprocess

import "log"

func PostProcessHTML(filename string, content string) error {
	log.Printf("Need to write: %s (%v bytes)\n", filename, len(content))

	return nil
}
