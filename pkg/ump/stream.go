package ump

import "strconv"

func (s *Stream) Fps() float64 {
	fps := s.R_frame_rate
	return fpsToFloat(fps)
}

func (s *Stream) DurationSeconds() float64 {
	d, err := strconv.ParseFloat(s.Duration, 64)
	if err != nil {
		return -1
	}
	di := int(d * 1000)
	d = float64(di) / 1000
	return d
}
