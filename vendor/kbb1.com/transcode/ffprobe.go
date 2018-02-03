package transcode

import (
	"bytes"
	"os/exec"
	"strings"
)

var (
	showargs = "-v quiet -print_format json -show_format -show_streams"
)

func ShowFormat(path string) (error, []byte) {
	params := strings.Fields(showargs)
	params = append(params, path)

	cmd := exec.Command("ffprobe", params...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Start()
	if err != nil {
		return err, nil
	}
	err = cmd.Wait()
	return err, out.Bytes()
}
