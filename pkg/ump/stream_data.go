package ump

import (
	"fmt"
	"os"
)

type GroupProfile struct {
	Files                     []string
	Info                      map[string]StreamInfo
	IntendedDurationProposals []float64
	ChannelLayoutMap          map[string]int
}

type StreamInfo struct {
	File           string
	CodecType      string
	StreamNumber   int
	StreamPosition int
	Width          int
	Height         int
	FPS            float64
	Duration       float64
	ChannelLayout  string
	Channels       int
	TranscodeNotes []string
}

func CollectGroupInfo(paths ...string) (GroupProfile, error) {
	gp := GroupProfile{}
	gp.Files = paths
	gp.Info = make(map[string]StreamInfo)
	gp.ChannelLayoutMap = make(map[string]int)
	if err := pathCheck(paths...); err != nil {
		return gp, fmt.Errorf("paths validation failed: %v", err)
	}
	for fileNumber, path := range paths {
		mp := NewProfile()
		if err := mp.ConsumeFile(path); err != nil {
			return gp, fmt.Errorf("failed to consume file %v: %v", path, err)
		}
		vmet, amet, smet := 0, 0, 0
		for n, stream := range mp.Streams {
			tag := ""
			si := StreamInfo{}
			si.File = path
			si.StreamNumber = n
			si.CodecType = stream.Codec_type
			si.Duration = stream.DurationSeconds()
			switch stream.Codec_type {
			case CODEC_TYPE_VIDEO:
				tag = "v"
				si.Width = stream.Width
				si.Height = stream.Height

				si.FPS = stream.Fps()
				gp.IntendedDurationProposals = append(gp.IntendedDurationProposals, (si.Duration * (si.FPS / 25.0)))
				si.StreamPosition = vmet
				vmet++
			case CODEC_TYPE_AUDIO:
				tag = "a"
				si.Channels = stream.Channels
				si.ChannelLayout = stream.Channel_layout
				gp.ChannelLayoutMap[si.ChannelLayout]++
				si.StreamPosition = amet
				amet++
			case CODEC_TYPE_SUBTITLE:
				tag = "s"
				si.StreamPosition = smet
				smet++
			default:
				continue
			}
			key := fmt.Sprintf("%v:%v:%v", fileNumber, tag, si.StreamPosition)
			gp.Info[key] = si
		}
	}
	return gp, nil
}

func pathCheck(paths ...string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no paths provided")
	}
	for i, path := range paths {
		f, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to check path %v (%v): %v", i, path, err)
		}
		if f.IsDir() {
			return fmt.Errorf("path %v (%v) is directory", i, path)
		}
	}
	return nil
}
