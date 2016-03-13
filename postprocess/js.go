package postprocess

import "log"

func PostProcessJS(filename string, content string) error {
	log.Printf("Need to write: %s (%v bytes)\n", filename, len(content))
	return nil
}
