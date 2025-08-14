package mediasource

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Galdoba/lazyam/pkg/ump"
)

type SourceFile struct {
	Ignore    bool     `json:"ignore"`
	Name      string   `json:"file"`
	Type      string   `json:"purpose"`
	Languages []string `json:"languages"`
	Layout    []string `json:"layout"`
	Interlace float64  `json:"interlace factor"`
	BitRate   []string `json:"audio bitrates"`
}

func NewSourceMedia(mp *ump.MediaProfile) SourceFile {
	sf := SourceFile{}
	sf.Name = filepath.Base(mp.Format.Filename)
	if strings.HasSuffix(sf.Name, ".srt") {
		sf.Type = "SRT"
	}
	count := 0
	for _, stream := range mp.Streams {
		count++
		switch stream.Codec_type {
		case ump.CODEC_TYPE_VIDEO:
			sf.Interlace = -1
			sf.Type = "SOURCE"
			fmt.Println("set source", sf.Name)
		case ump.CODEC_TYPE_AUDIO:
			switch stream.Channels {
			case 2:
				sf.Layout = append(sf.Layout, "20")
			case 6:
				sf.Layout = append(sf.Layout, "51")
			}
			sf.Languages = append(sf.Languages, stream.Tags["language"])
			sf.BitRate = append(sf.BitRate, stream.Bit_rate)
		case ump.CODEC_TYPE_SUBTITLE:
		}
	}
	if count == 0 {
		// fmt.Println("ignore", sf.Name)
		sf.Ignore = true
	}
	return sf
}
