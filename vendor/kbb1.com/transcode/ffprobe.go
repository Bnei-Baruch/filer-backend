package transcode

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
)

type (
	FFstream struct {
		Codec     string `json:"codec_name"`
		Type      string `json:"codec_type"`
		FrameRate string `json:"r_frame_rate"`
		BitRate   int64  `json:"bit_rate,string"`
		Channels  int64  `json:"channels"`
		Width     int64  `json:"width"`
		Height    int64  `json:"height"`
	}

	FFformat struct {
		Name     string  `json:"format_name"`
		Duration float64 `json:"duration,string"`
		BitRate  int64   `json:"bit_rate,string"`
	}

	FFprobe struct {
		Format  FFformat    `json:"format", mapstructure:"format"`
		Streams []*FFstream `json:"streams, mapstructure:"streams"`
	}
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

func Probe(path string) (error, *FFprobe) {
	err, format := ShowFormat(path)
	if err == nil {
		probe := new(FFprobe)
		err = json.Unmarshal(format, probe)
		if err == nil {
			return err, probe
		}
	}
	return err, nil
}
