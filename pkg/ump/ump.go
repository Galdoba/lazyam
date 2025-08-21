package ump

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Galdoba/devtools/cli/command"
)

const (
	ScanBasic     = "basic"
	ScanInterlace = "interlace"
	ScanSilence   = "silence"
	Status_OK     = "OK"
	Status_Error  = "Error"

	CODEC_TYPE_VIDEO    = "video"
	CODEC_TYPE_AUDIO    = "audio"
	CODEC_TYPE_SUBTITLE = "subtitle"
)

func NewProfile() *MediaProfile {
	sr := &MediaProfile{}
	//	sr.Format = &Format{}
	return sr
}

func (prof *MediaProfile) ConsumeFile(path string) error {
	stdout, stderr, err := command.Execute("ffprobe "+fmt.Sprintf("-v quiet -print_format json -show_format -show_streams -show_programs %v", path), command.Set(command.BUFFER_ON))
	if err != nil {
		if err.Error() != "exit status 1" {
			return fmt.Errorf("execution error: %v", err.Error())
		}
	}
	if stderr != "" {
		fmt.Println("stderr:")
		fmt.Println(stderr)
		panic("неожиданный выхлоп")
		//
	}
	data := []byte(stdout)
	if len(data) == 0 {
		flbts, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("file reading error: %v", err.Error())
		}
		if len(flbts) == 0 {
			return fmt.Errorf("file empty: %v", path)
		}
		check, _ := command.New(
			command.CommandLineArguments("ffprobe", fmt.Sprintf("-hide_banner "+fmt.Sprintf("-i %v", path))),
			//command.Set(command.TERMINAL_ON),
			command.Set(command.BUFFER_ON),
		)
		check.Run()
		checkOut := check.StdOut() + check.StdErr()
		if checkOut != "" {
			return fmt.Errorf("can't read: %v", checkOut)
		}
	}
	err = json.Unmarshal(data, &prof)
	if err != nil {
		return fmt.Errorf("can't unmarshal data from file: %v (%v)\n%v", err.Error(), path, string(data))
	}
	err = prof.validate()
	if err != nil {
		return fmt.Errorf("validation error: %v", err.Error())
	}
	//fmt.Println(prof.Short())

	return nil
}

// func (prof *MediaProfile) ConsumeJSON(path string) error {
// 	if !strings.HasSuffix(path, ".json") {
// 		return fmt.Errorf("file is not json")
// 	}
// 	data, err := os.ReadFile(path)
// 	if err != nil {
// 		return fmt.Errorf("can't read json: %v", err.Error())
// 	}
// 	err = json.Unmarshal(data, &prof)
// 	if err != nil {
// 		return fmt.Errorf("can't unmarshal json: %v (%v)", err.Error(), path)
// 	}
// 	err = prof.validate()
// 	if err != nil {
// 		return fmt.Errorf("validation error: %v", err.Error())
// 	}
// 	return nil
// }

// func MapStorage(dir string) map[string]*MediaProfile {
// 	fls, _ := os.ReadDir(dir)
// 	prfMap := make(map[string]*MediaProfile)
// 	for _, fl := range fls {
// 		if strings.HasSuffix(fl.Name(), ".json") {
// 			mp := NewProfile()
// 			mp.ConsumeJSON(dir + fl.Name())
// 			key := filepath.Base(mp.Format.Filename)
// 			prfMap[key] = mp
// 		}
// 	}
// 	return prfMap
// }

func (mp *MediaProfile) MarshalJSON() ([]byte, error) {
	type MediaProfileAlias MediaProfile
	return json.MarshalIndent(&struct {
		*MediaProfileAlias
	}{
		MediaProfileAlias: (*MediaProfileAlias)(mp),
	}, "", "  ")
}

// func (mp *MediaProfile) SaveAs(path string) error {
// 	bt, err := mp.MarshalJSON()
// 	if err != nil {
// 		return err
// 	}
// 	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
// 	if err != nil {
// 		return fmt.Errorf("can't save target file: %v", err.Error())
// 	}
// 	defer f.Close()
// 	f.Truncate(0)
// 	_, err = f.Write(bt)
// 	return nil
// }

func assertNoError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

type MediaProfile struct {
	BasicStatus      string    `json:"Basic Check,omitempty"`
	RDWR_Status      string    `json:"Read-Write Check,omitempty"`
	Interlace_Status string    `json:"Interlace Check,omitempty"`
	Format           *Format   `json:"format"`
	Streams          []*Stream `json:"streams,omitempty"`

	warnings       []string
	streamInfo     map[string]string
	short          string
	long           string
	chanLayout     string
	ScansCompleted []string `json:"scans completed,omitempty"`
}

// type MediaProfile interface {
// 	Warnings() []string
// 	Short() string
// 	Long() string
// 	AudioLayout() string
// 	ConfirmScan(string) error
// 	ConsumeFile(string) error
// 	ScanBasic(string) error
// }

var ErrNoScanNeeded = errors.New("no scan needed")

// func (mp *MediaProfile) ScanBasic(sourceFile string) error {
// 	switch mp.BasicStatus {
// 	case Status_OK:
// 		return ErrNoScanNeeded
// 	}
// 	if strings.Contains(sourceFile, " ") {
// 		return fmt.Errorf("can't perform scan: filepath contains space")
// 	}

// 	if mp.scanCompleted(ScanBasic) {
// 		return fmt.Errorf("can't perform scan: scan was already completed")
// 	}
// 	err := mp.ConsumeFile(sourceFile)
// 	if err != nil {
// 		return err
// 	}
// 	if mp.confirmScan(ScanBasic) != nil {
// 		return err
// 	}
// 	mp.BasicStatus = Status_OK
// 	return nil
// }

// func (mp *MediaProfile) ScanInterlace(sourceFile string) error {
// 	if mp.scanCompleted(ScanInterlace) {
// 		return fmt.Errorf("can't perform scan: %v scan was already completed", ScanInterlace)
// 	}
// 	if !mp.scanCompleted(ScanBasic) {
// 		return fmt.Errorf("can't perform scan: basic scan is required for interlace scan")
// 	}
// 	frames := 9999
// 	//COMENCE INTERLACE SCAN
// 	devnull := ""
// 	switch runtime.GOOS {
// 	case "linux":
// 		devnull = "/dev/null"
// 	case "windows":
// 		devnull = "NUL"
// 	}
// 	if _, ok := mp.streamInfo["0:v:0"]; !ok {
// 		return fmt.Errorf("can't perform interlace scan: no video streams detected")
// 	}

// 	com := fmt.Sprintf("ffmpeg -hide_banner -filter:v idet -frames:v %v -an -f rawvideo -y %v -i %v", frames, devnull, sourceFile)
// 	fmt.Fprintf(os.Stderr, "run: %v\n", com)

// 	done := false
// 	var wg sync.WaitGroup
// 	process, err := command.New(command.CommandLineArguments(com),
// 		command.AddBuffer("buf"),
// 	)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "subprocess error1: %v", err.Error())
// 		return err
// 	}
// 	buf := process.Buffer("buf")
// 	wg.Add(1)
// 	go func() {
// 		err = process.Run()
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "subprocess error2: %v", err.Error())
// 		}
// 		done = true
// 		wg.Done()
// 	}()
// 	for !done {
// 		time.Sleep(time.Millisecond * 500)
// 		//bts, _ := os.ReadFile("aaa.txt")

// 		ln := strings.Split(buf.String(), "\n")
// 		last := len(ln) - 1
// 		if last < 0 {
// 			last = 0
// 		}
// 		if strings.Contains(ln[last], "s/s speed=") {
// 			fmt.Fprintf(os.Stderr, "%v\r", ln[last])
// 		}
// 	}
// 	fmt.Fprintf(os.Stderr, "\n")
// 	wg.Wait()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "subprocess error3: %v", err.Error())
// 		return err
// 	}

// 	idetReport := filterIdet(buf.String())
// 	sum := float64(idetReport["I"] + idetReport["P"])
// 	progressive := float64(idetReport["P"]) / sum
// 	progressive = float64(int(progressive*10000)) / 100
// 	mp.Streams[0].Progressive_frames_pct = progressive

// 	if mp.confirmScan(ScanInterlace) != nil {
// 		return err
// 	}
// 	return nil
// }

// func (mp *MediaProfile) ScanSilence(sourceFile string, dB int, duration float64) error {
// 	if mp.scanCompleted(ScanSilence) {
// 		return fmt.Errorf("can't scan: %v scan was already completed", ScanInterlace)
// 	}
// 	if !mp.scanCompleted(ScanBasic) {
// 		return fmt.Errorf("can't scan: basic scan required for interlace scan")
// 	}
// 	//COMENCE SILENCE SCAN

// 	com := fmt.Sprintf("ffmpeg -hide_banner -i %v -af silencedetect=n=-%vdB:d=%v -f null -", sourceFile, dB, duration)
// 	fmt.Fprintf(os.Stderr, "run: '%v'\n", com)

// 	done := false
// 	var wg sync.WaitGroup
// 	process, err := command.New(command.CommandLineArguments(com),
// 		command.AddBuffer("buf"),
// 		// command.Set(command.BUFFER_ON),
// 	)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "subprocess error1: %v", err.Error())
// 		return err
// 	}
// 	buf := process.Buffer("buf")
// 	wg.Add(1)
// 	go func() {
// 		err = process.Run()
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "subprocess error2: %v", err.Error())
// 		}
// 		done = true
// 		wg.Done()
// 	}()
// 	for !done {
// 		ln := strings.Split(buf.String(), "\n")
// 		last := len(ln) - 1
// 		if last < 0 {
// 			last = 0
// 		}

// 		out, _ := filterSilence(buf.String())
// 		fmt.Fprintf(os.Stderr, "%v  \r", out)

// 	}
// 	fmt.Fprintf(os.Stderr, "                                            \r")
// 	wg.Wait()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "subprocess error3: %v", err.Error())
// 		return err
// 	}
// 	report, silenceData := filterSilence(buf.String())
// 	mp.Streams[0].SilenceData = silenceData
// 	fmt.Fprintf(os.Stdout, report+"\n")
// 	if mp.confirmScan(ScanSilence) != nil {
// 		return err
// 	}
// 	return nil
// }

// func filterSilence(report string) (string, []SilenceSegment) {
// 	silenceData := []SilenceSegment{}
// 	lines := strings.Split(report, "\n")
// 	total := 0.0
// 	timestamp := ""
// 	for _, line := range lines {
// 		if strings.Contains(line, "silence_end") {
// 			sil := SilenceSegment{}
// 			flds := strings.Fields(line)
// 			if flds[3] == "silence_end:" {
// 				end, err := strconv.ParseFloat(flds[4], 64)
// 				if err != nil {
// 					fmt.Println(line)
// 					panic(err.Error() + line)
// 				}
// 				sil.SilenceEnd = end
// 			}
// 			if flds[6] == "silence_duration:" {
// 				dur, err := strconv.ParseFloat(flds[7], 64)
// 				if err != nil {
// 					fmt.Println(line)
// 					panic(err.Error() + line)
// 				}
// 				sil.SilenceDuration = dur
// 			}

// 			sil.SilenceStart = sil.SilenceEnd - sil.SilenceDuration
// 			if sil.SilenceStart < 0 {
// 				sil.SilenceStart = 0
// 			}
// 			total += sil.SilenceDuration
// 			silenceData = append(silenceData, sil)
// 		}
// 		if strings.Contains(line, "time=") {
// 			data := strings.Fields(line)
// 			for _, time := range data {
// 				if strings.HasPrefix(line, "time=") {
// 					time = strings.TrimPrefix(time, "time=")
// 					timestamp = time
// 				}
// 			}
// 		}
// 	}
// 	total = float64(int(total*1000)) / 1000
// 	return fmt.Sprintf("%v: %v seconds of silence in %v segments detected", timestamp, total, len(silenceData)), silenceData
// }

// func filterIdet(report string) map[string]int {
// 	rMap := make(map[string]int)
// 	lines := strings.Split(report, "\n")
// 	for _, ln := range lines {
// 		if !strings.Contains(ln, "Parsed_idet") {
// 			continue
// 		}
// 		if strings.Contains(ln, "Repeated ") {
// 			words := strings.Split(ln, " ")
// 			fn := 0
// 			for _, w := range words {
// 				n, err := strconv.Atoi(w)
// 				if err != nil {
// 					continue
// 				}
// 				fn++
// 				rMap["T"] += n
// 				switch fn {
// 				case 2, 3:
// 					rMap["I"] += n
// 				}
// 			}
// 		}
// 		if strings.Contains(ln, "Single ") {
// 			words := strings.Split(ln, " ")
// 			fn := 0
// 			for _, w := range words {
// 				n, err := strconv.Atoi(w)
// 				if err != nil {
// 					continue
// 				}
// 				fn++
// 				rMap["T"] += n
// 				switch fn {
// 				case 1, 2:
// 					rMap["I"] += n
// 				case 3:
// 					rMap["P"] += n
// 				}
// 			}
// 		}
// 		if strings.Contains(ln, "Multi ") {
// 			words := strings.Split(ln, " ")
// 			fn := 0
// 			for _, w := range words {
// 				n, err := strconv.Atoi(w)
// 				if err != nil {
// 					continue
// 				}
// 				fn++
// 				rMap["T"] += n
// 				switch fn {
// 				case 1, 2:
// 					rMap["I"] += n
// 				case 3:
// 					rMap["P"] += n
// 				}
// 			}
// 		}
// 	}
// 	return rMap
// }

// func (mp *MediaProfile) scanCompleted(scanType string) bool {
// 	for _, scanned := range mp.ScansCompleted {
// 		if scanType == scanned {
// 			return true
// 		}
// 	}
// 	return false
// }

/*
TZ:
Утилита должна оценить файл и составить его рабочий Профиль

//
Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'Barri_4s_treyler_a_teka.mp4':

	Duration: 00:01:26.08, start: 0.000000, bitrate: 15338 kb/s
	  Stream #0:0(eng): Video: h264 (Main) (avc1 / 0x31637661), yuv420p, 1920x1080 [SAR 1:1 DAR 16:9], 14831 kb/s, 25 fps, 25 tbr, 25k tbn, 50 tbc (default)
	  Stream #0:1(eng): Audio: aac (LC) (mp4a / 0x6134706D), 48000 Hz, stereo, fltp, 317 kb/s (default)

//
[1100]
[1{#1920x1080#25#5822#[]#[[SAR 1:1 DAR 16:9]]#5.3%};2{#5.1#48.0#341}{#stereo#48.0#129};0;1]

ffprobe -v quiet -of json -show_format -show_streams Barri_4s_treyler_a_teka.mp4




Структура Профиля:
краткий:
[ABCDE-F], где
a = количество видеопотоков
b = количество аудиопотоков
c = количество потоков данных
d = количество srt потоков
e = количество хз чего
f = количество замечаний от библиотеки парса

Сепараторы:
- = логический раздел (Абзац)
; = потоки (Строка)
# = данные внутри потоков (Слово)

развернутый:
[A#a#b#c#d#e#f;B#g#h#i;C          ], где
A = количество видеопотоков     формат: eHex                (eHex = целое число с базой 32)          int
a = размер видео                формат: IxI                 (I = целое число)                        int
b = fps видео                   формат: F                   (F = десятичное число)                   float64
c = битрейт                     формат: I                   (I = целое число)                        int
d = SARDAR внешний              формат: [SAR I:I DAR I:I]   (I = целое число)                        []int
e = SARDAR внутренний           формат: [d]                 (d = формат внутреннего SARDAR)          []int
f = интерлейс                   формат: F                   (F = средний % не прогрессивных кадров)  float64
B = количество аудиопотоков     формат: eHex                (eHex = целое число с базой 32)          int
g = раскладка каналов           формат: S                   (S = текст ключ-значение по таблице)     string
h = герцовка                    формат: F                   (F = десятичное число kHz)               float64
i = битрейт                     формат: I                   (I = целое число значение kbit/s)        int
C = количество потоков данных   формат: eHex                (eHex = целое число с базой 32)          int??????????

e = битрейт      формат: [I]   (I = целое число)
d = количество srt потоков
e = количество хз чего
f = количество замечаний от библиотеки парса

Определения:
1.Профиль - форматированная информация о внутренней медиа структуре файла.
2.Аргументы = файлы к которым нужно составить профайл
*/

// validate - проверяет данные на целостность и формирует отчетные строки
func (p *MediaProfile) validate() error {
	vSNum := 0
	aSNum := 0
	dSNum := 0
	sSNum := 0
	p.streamInfo = make(map[string]string)
	for _, stream := range p.Streams {
		switch stream.Codec_type {
		default:
			return fmt.Errorf("unknown codec type: %v", stream.Codec_type)
		case "video":
			p.validateVideo(stream, vSNum)
			vSNum++
		case "audio":
			p.validateAudio(stream, aSNum)
			aSNum++
		case "data":
			dSNum++
		case "subtitle":
			sSNum++
		}
	}

	switch vSNum {
	case 1:
	case 0:
		if aSNum+sSNum == 0 {
			p.warnings = append(p.warnings, fmt.Sprintf("file ==> no video, audio and subtitle streams detected"))
		}
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("file ==> %v video streams detected", vSNum))
	}
	switch aSNum {
	case 0:
	case 1, 2:
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("file ==> %v audio streams detected", aSNum))
	}
	switch sSNum {
	case 0:
	case 1:
		p.warnings = append(p.warnings, fmt.Sprintf("file ==> %v subtitle stream detected", sSNum))
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("file ==> %v subtitle streams detected", sSNum))
	}
	p.short = fmt.Sprintf("%v%v%v%v-%v", ehex(vSNum), ehex(aSNum), ehex(dSNum), ehex(sSNum), ehex(len(p.warnings)))
	p.combineLong(vSNum, aSNum, dSNum, sSNum)
	p.chanLayout = strings.TrimSuffix(p.chanLayout, " ")
	return nil
}

func (p *MediaProfile) validateVideo(stream *Stream, vSNum int) {
	currentBlock := fmt.Sprintf("0:v:%v", vSNum)
	vSize := fmt.Sprintf("%vx%v", stream.Width, stream.Height)
	switch vSize {
	case "720x576":
		vSize = "SD"
	case "1920x1080":
		vSize = "HD"
	case "3840x2160":
		vSize = "4K"
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> bad video size: [%v]", vSNum, vSize))
		vSize = "[" + vSize + "]"
	}
	p.streamInfo[currentBlock] = "#" + vSize

	fps := stream.R_frame_rate
	fps_block := fpsToFloat(fps)
	switch fps {
	case "24/1":
	case "25/1":
	case "24000/1001":
	case "2997/125": //about valid
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> bad fps: [%v]", vSNum, fps_block))
	}
	p.streamInfo[currentBlock] += "#" + fmt.Sprintf("%v", fps_block)

	sardar := "SAR=" + stream.Sample_aspect_ratio + " DAR=" + stream.Display_aspect_ratio
	sardar = strings.ReplaceAll(sardar, " ", "_")
	switch sardar {
	case "SAR=1:1_DAR=16:9", "":
	case "SAR=_DAR=":
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> blank SAR DAR data present", vSNum))
		sardar = "???"
	case "SAR=1:1_DAR=1024:429", "SAR=1:1_DAR=37:20", "SAR=1:1_DAR=160:67":
		//p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> bad (but possible) SAR DAR: [%v]", vSNum, sardar))
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> bad SAR DAR: [%v]", vSNum, sardar))
	}

	p.streamInfo[currentBlock] += "#" + fmt.Sprintf("[%v]", sardar)

	btrate := stream.Bit_rate
	switch btrate {
	case "":
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> blank bitRate data present", vSNum))
		btrate = "???"
	default:
		btr, err := strconv.Atoi(btrate)
		if err != nil {
			panic(err.Error())
		}
		if btr == 0 {
			panic(fmt.Sprintf("stream [0:v:%v] ==> bitRate: [%v]", vSNum, btr))
		}
		//p.warnings = append(p.warnings, fmt.Sprintf("stream [0:v:%v] ==> bitRate: [%v]", vSNum, btr))
		btrate = fmt.Sprintf("%v", btr/1000)
	}
	p.streamInfo[currentBlock] += "#" + fmt.Sprintf("%v", btrate)
	prog := stream.Progressive_frames_pct
	switch prog {
	case 0.0:
		p.streamInfo[currentBlock] += `#ns`
	default:
		p.streamInfo[currentBlock] += `#` + fmt.Sprintf("%v", prog) + "%"
		if prog < 95 && prog > 0 {
			text := fmt.Sprintf("stream [0:v:%v] ==> interlace suspected (%v", vSNum, 100-prog) + "%)"
			p.warnings = append(p.warnings, text)
		}
	}
}

func (p *MediaProfile) validateAudio(stream *Stream, aSNum int) {
	currentBlock := fmt.Sprintf("0:a:%v", aSNum)
	chan_lay := stream.Channel_layout
	channel_num := stream.Channels
	switch chan_lay {
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> unknown channel layout provided (%v)", aSNum, chan_lay))
		chan_lay = chan_lay + ":" + fmt.Sprintf("%vch", channel_num)
	case "":
		switch stream.Channels {
		default:
			p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> no channel layout provided (can not guess: %v channels)", aSNum, channel_num))
			chan_lay = "???"
		case 1:
			chan_lay = "*mono"
		case 2:
			chan_lay = "*stereo"
		case 6:
			chan_lay = "*5.1"
		}
	case "1 channels (FL)", "1 channels (LFE)", "1 channels (BL)", "1 channels (FR)", "1 channels (BR)", "5 channels (FL+FR+LFE+SL+SR)", "downmix":
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> unusual channel layout provided (%v)", aSNum, chan_lay))
		chan_lay = "*:" + fmt.Sprintf("%vch", channel_num)
	case "5.1":
	case "5.1(side)":
		chan_lay = "5.1"
	case "mono":
	case "stereo":
	}
	p.streamInfo[currentBlock] += "#" + fmt.Sprintf("%v", chan_lay)

	p.chanLayout += fmt.Sprintf("%v ", ehex(channel_num))

	switch channel_num {
	case 1, 2, 6:
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> unusual number of channels (%v)", aSNum, channel_num))
		//p.streamInfo[currentBlock] += ":" + fmt.Sprintf("%vch", ehex(channel_num))
	}

	hz := stream.Sample_rate
	switch hz {
	default:
		p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> sample rate [%v Hz]", aSNum, hz))
	case "48000":
	}
	p.streamInfo[currentBlock] += "#" + hzFormat(hz)

	bitRt := stream.Bit_rate
	bits, err := strconv.Atoi(bitRt)
	if err != nil {
		switch bitRt {
		case "":
			p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> no bitrate provided", aSNum))
			p.streamInfo[currentBlock] += "#???"
			return
		}
		if bitRt != "" {
			//panic(fmt.Sprintf("stream [0:a:%v] ==> bad bitrate  (%v): %v", aSNum, bitRt, err.Error()))
			p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> bad bitrate  (%v): %v", aSNum, bitRt, err.Error()))
		}
	}
	switch bits {
	case 0:
	default:
		if bits < 80000 {
			p.warnings = append(p.warnings, fmt.Sprintf("stream [0:a:%v] ==> silence suspected: bitrate is extreamly low [%v b/s]", aSNum, bitRt))
		}
	}
	p.streamInfo[currentBlock] += "#" + fmt.Sprintf("%v", bits/1000)
}

func (p *MediaProfile) combineLong(vSNum, aSNum, dSNum, sSNum int) {
	p.long = ""
	for _, stTp := range []string{"v", "a", "d", "s"} {
		switch stTp {
		case "v":
			p.long += fmt.Sprintf("%v", vSNum)
		case "a":
			p.long += fmt.Sprintf("%v", aSNum)
		case "d":
			p.long += fmt.Sprintf("%v", dSNum)
		case "s":
			p.long += fmt.Sprintf("%v", sSNum)
		}
		//[1{#1920x1080#25#5822#[]#[[SAR 1:1 DAR 16:9]]#5.3%};2{#5.1#48.0#341}{#stereo#48.0#129};0;1]
		for i := 0; i < 50; i++ {
			if val, ok := p.streamInfo[fmt.Sprintf("0:%v:%v", stTp, i)]; ok {
				p.long += fmt.Sprintf("%v%v{%v}", stTp, i, val)
			}
		}
		p.long += ";"
	}
	//p.long = strings.TrimSuffix(p.long, ";")
	p.long += fmt.Sprintf("w%v", len(p.warnings))
	p.long = strings.ReplaceAll(p.long, " ", "_")
}

func ehex(i int) string {
	switch i {
	default:
		return "?"
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
		return fmt.Sprintf("%v", i)
	case 10:
		return "A"
	case 11:
		return "B"
	case 12:
		return "C"
	case 13:
		return "D"
	case 14:
		return "E"
	case 15:
		return "F"
	case 16:
		return "G"
	case 17:
		return "H"
	case 18:
		return "J"
	case 19:
		return "K"
	case 20:
		return "L"
	case 21:
		return "M"
	case 22:
		return "N"
	case 23:
		return "P"
	case 24:
		return "Q"
	case 25:
		return "R"
	case 26:
		return "S"
	case 27:
		return "T"
	case 28:
		return "U"
	case 29:
		return "V"
	case 30:
		return "W"
	case 31:
		return "X"
	case 32:
		return "Y"
	case 33:
		return "Z"
	}
}

func (p *MediaProfile) Warnings() []string {
	return p.warnings
}

func (p *MediaProfile) Short() string {
	return p.short
}

func (p *MediaProfile) Long() string {
	return p.long
}

func (p *MediaProfile) ByStream() map[string]string {
	return p.streamInfo
}

func (p *MediaProfile) AudioLayout() string {
	return p.chanLayout
}

func fpsToFloat(fps string) float64 {
	data := strings.Split(fps, "/")
	if len(data) != 2 {
		return -1
	}
	i1, err := strconv.Atoi(data[0])
	if err != nil {
		return -1
	}
	i2, err := strconv.Atoi(data[1])
	if err != nil {
		return -1
	}
	fl := float64(i1) / float64(i2)
	fli := float64(int(fl*1000)) / 1000
	return fli
}

func hzFormat(hz string) string {
	h, err := strconv.Atoi(hz)
	if err != nil {
		return "???"
	}
	return fmt.Sprintf("%v", float64(h)/1000)
}

// func (mp *MediaProfile) confirmScan(scan string) error {
// 	if mp.scanCompleted(scan) {
// 		return fmt.Errorf("confirmation failed: was already confirmed")
// 	}
// 	switch scan {
// 	default:
// 		return fmt.Errorf("can't confirm %v scan: unknown or unimplemented scan type")
// 	case ScanBasic:
// 		if len(mp.ScansCompleted) != 0 {
// 			return fmt.Errorf("can't confirm %v scan: other scan data must not exist")
// 		}
// 	case ScanInterlace, ScanSilence:
// 		if !mp.scanCompleted(ScanBasic) {
// 			return fmt.Errorf("can't confirm %v scan: basic scan not completed")
// 		}

// 	}
// 	if err := mp.validate(); err != nil {
// 		return fmt.Errorf("can't validate profile: %v", err.Error())
// 	}
// 	mp.ScansCompleted = appendUnique(mp.ScansCompleted, scan)
// 	return nil
// }

func appendUnique(slice []string, newElem string) []string {
	for i := range slice {
		if slice[i] == newElem {
			return slice
		}
	}
	return append(slice, newElem)
}

// func (sr *MediaProfile) String() string {
// 	s := "file: " + sr.Format.Filename + "\n"
// 	s += fmt.Sprintf("streams total: %v", len(sr.Streams))
// 	return s
// }

type Format struct {
	Bit_rate         string            `json:"bit_rate,omitempty"`
	Duration         string            `json:"duration,omitempty"`
	Filename         string            `json:"filename,omitempty"`
	Format_long_name string            `json:"format_long_name,omitempty"`
	Format_name      string            `json:"format_name,omitempty"`
	Nb_programs      int               `json:"nb_programs,omitempty"`
	Nb_streams       int               `json:"nb_streams,omitempty"`
	Probe_score      int               `json:"probe_score,omitempty"`
	Size             string            `json:"size,omitempty"`
	Start_time       string            `json:"start_time,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

type Stream struct {
	Avg_frame_rate         string                  `json:"avg_frame_rate,omitempty"`
	Bit_rate               string                  `json:"bit_rate,omitempty"`
	Bits_per_raw_sample    string                  `json:"bits_per_raw_sample,omitempty"`
	Bits_per_sample        int                     `json:"bits_per_sample,omitempty"`
	Channel_layout         string                  `json:"channel_layout,omitempty"`
	Channels               int                     `json:"channels,omitempty"`
	Chroma_location        string                  `json:"chroma_location,omitempty"`
	Closed_captions        int                     `json:"closed_captions,omitempty"`
	Codec_long_name        string                  `json:"codec_long_name,omitempty"`
	Codec_name             string                  `json:"codec_name,omitempty"`
	Codec_tag              string                  `json:"codec_tag,omitempty"`
	Codec_tag_string       string                  `json:"codec_tag_string,omitempty"`
	Codec_time_base        string                  `json:"codec_time_base,omitempty"`
	Codec_type             string                  `json:"codec_type,omitempty"`
	Coded_height           int                     `json:"coded_height,omitempty"`
	Coded_width            int                     `json:"coded_width,omitempty"`
	Color_primaries        string                  `json:"color_primaries,omitempty"`
	Color_range            string                  `json:"color_range,omitempty"`
	Color_space            string                  `json:"color_space,omitempty"`
	Color_transfer         string                  `json:"color_transfer,omitempty"`
	Display_aspect_ratio   string                  `json:"display_aspect_ratio,omitempty"`
	Divx_packed            string                  `json:"divx_packed,omitempty"`
	Dmix_mode              string                  `json:"dmix_mode,omitempty"`
	Duration               string                  `json:"duration,omitempty"`
	Duration_ts            int                     `json:"duration_ts,omitempty"`
	Field_order            string                  `json:"field_order,omitempty"`
	Has_b_frames           int                     `json:"has_b_frames,omitempty"`
	Height                 int                     `json:"height,omitempty"`
	Id                     string                  `json:"id,omitempty"`
	Index                  int                     `json:"index,omitempty"`
	Is_avc                 string                  `json:"is_avc,omitempty"`
	Level                  int                     `json:"level,omitempty"`
	Loro_cmixlev           string                  `json:"loro_cmixlev,omitempty"`
	Loro_surmixlev         string                  `json:"loro_surmixlev,omitempty"`
	Ltrt_cmixlev           string                  `json:"ltrt_cmixlev,omitempty"`
	Ltrt_surmixlev         string                  `json:"ltrt_surmixlev,omitempty"`
	Max_bit_rate           string                  `json:"max_bit_rate,omitempty"`
	Nal_length_size        string                  `json:"nal_length_size,omitempty"`
	Nb_frames              string                  `json:"nb_frames,omitempty"`
	Pix_fmt                string                  `json:"pix_fmt,omitempty"`
	Profile                string                  `json:"profile,omitempty"`
	Progressive_frames_pct float64                 `json:"progressive_frames_pct,omitempty"`
	Quarter_sample         string                  `json:"quarter_sample,omitempty"`
	R_frame_rate           string                  `json:"r_frame_rate,omitempty"`
	Refs                   int                     `json:"refs,omitempty"`
	Sample_aspect_ratio    string                  `json:"sample_aspect_ratio,omitempty"`
	Sample_fmt             string                  `json:"sample_fmt,omitempty"`
	Sample_rate            string                  `json:"sample_rate,omitempty"`
	SilenceData            []SilenceSegment        `json:"silence_segments,omitempty"`
	Start_pts              int                     `json:"start_pts,omitempty"`
	Start_time             string                  `json:"start_time,omitempty"`
	Time_base              string                  `json:"time_base,omitempty"`
	Width                  int                     `json:"width,omitempty"`
	Side_data_list         []Side_data_list_struct `json:"side_data_list,omitempty"`
	Tags                   map[string]string       `json:"tags,omitempty"`
	Disposition            map[string]int          `json:"disposition,omitempty"`
}

type Side_data_list_struct struct {
	Side_data map[string]string
}

type SilenceSegment struct {
	SilenceStart    float64 `json:"start,omitempty"`
	SilenceEnd      float64 `json:"end,omitempty"`
	SilenceDuration float64 `json:"duration,omitempty"`
	LoudnessBorder  float64 `json:"loudness_border,omitempty"`
}

/*
Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'Shifter_5.1_RUS.mov':
  Duration: 00:01:17.52, start: 0.000000, bitrate: 174896 kb/s
    Stream #0:0(eng): Video: prores (HQ) (apch / 0x68637061), yuv422p10le(tv, bt709, progressive), 1920x1080, 167876 kb/s, SAR 1:1 DAR 16:9, 25 fps, 25 tbr, 25 tbn, 25 tbc (default)
    Stream #0:1(eng): Audio: pcm_s24le (lpcm / 0x6D63706C), 48000 Hz, 5.1, s32 (24 bit), 6912 kb/s (default)
    Stream #0:2(eng): Data: none (tmcd / 0x64636D74) (default)


ffprobe -t 120 -f lavfi -i amovie=Mese_speyd_s01e03_PRT240129003542_SER_03970_18.mp4,asetnsamples=48000,astats=metadata=1:reset=1 -show_entries frame=pkt_pts_time:frame_tags=lavfi.astats.1.RMS_level,lavfi.astats.2.RMS_level,lavfi.astats.3.RMS_level,lavfi.astats.4.RMS_level,lavfi.astats.5.RMS_level,lavfi.astats.6.RMS_level,lavfi.astats.7.RMS_level,lavfi.astats.8.RMS_level,,lavfi.astats.1.8.RMS_level -of csv=p=0 1>log.txt

ffprobe -i INTERLACE_Zhivesh_tolko_raz--TRL--yolo_trl_hd_20_rus_18.mp4 -map 0:1 -show_entries frame=pkt_pts_time:frame_tags=lavfi.astats.1.RMS_level,lavfi.astats.2.RMS_level,lavfi.astats.3.RMS_level,lavfi.astats.4.RMS_level,lavfi.astats.5.RMS_level,lavfi.astats.6.RMS_level,lavfi.astats.7.RMS_level,lavfi.astats.7.RMS_level -of csv=p=0 1>log.txt

ffmpeg -t 60 -i Mese_speyd_s01e03_PRT240129003542_SER_03970_18.mp4 -map 0:a:1 -af "asetnsamples=48000,astats=reset=1:metadata=1,ametadata=print:key='lavfi.astats.OVERALL.RMS_level':file=stats1.log"  -f null -



ffprobe -f lavfi -i  amovie=Mese_speyd_s01e03_PRT240129003542_SER_03970_18.mp4,asetnsamples=48000*60,astats=metadata=1:reset=1 -show_entries frame=pkt_pts_time:frame_tags=lavfi.astats.1.RMS_level,lavfi.astats.2.RMS_level,lavfi.astats.3.RMS_level,lavfi.astats.4.RMS_level,lavfi.astats.5.RMS_level,lavfi.astats.6.RMS_level,lavfi.astats.7.RMS_level,lavfi.astats.8.RMS_level -of csv=p=0



ffmpeg -t 180 -y -i Mese_speyd_s01e03_PRT240129003542_SER_03970_18.mp4 -filter_complex "[0:a:0]pan=mono|c0=c0[ch0]; [0:a:0]pan=mono|c0=c1[ch1]; [0:a:0]pan=mono|c0=c2[ch2]; [0:a:0]pan=mono|c0=c3[ch3]; [0:a:0]pan=mono|c0=c4[ch4]; [0:a:0]pan=mono|c0=c5[ch5]; [0:a:1]pan=mono|c0=c0[ch6]; [0:a:1]pan=mono|c0=c1[ch7]; [ch0][ch1][ch2][ch3][ch4][ch5][ch6][ch7]amerge=inputs=9[out]" -map [out] -acodec alac audio_joined.m4a && ffprobe -f lavfi -i amovie=audio_joined.m4a,asetnsamples=48000*60,astats=metadata=1:reset=1 -show_entries frame=pkt_pts_time:frame_tags=lavfi.astats.1.RMS_level,lavfi.astats.2.RMS_level,lavfi.astats.3.RMS_level,lavfi.astats.4.RMS_level,lavfi.astats.5.RMS_level,lavfi.astats.6.RMS_level,lavfi.astats.7.RMS_level,lavfi.astats.8.RMS_level,lavfi.astats.9.RMS_level -of csv=p=0




fflite -t 180 -y -i Mese_speyd_s01e03_PRT240129003542_SER_03970_18.mp4  -map [left]      @alac0 left.m4a -map [right]     @alac0 right.m4a -map [center]    @alac0 center.m4a -map [lfe]       @alac0 lfe.m4a -map [left_sub]  @alac0 left_sub.m4a -map [right_sub] @alac0 right_sub.m4a


ffprobe -f lavfi -i  amovie=left.m4a,asetnsamples=48000*60,astats=metadata=1:reset=1 -show_entries frame=pkt_pts_time:frame_tags=lavfi.astats.1.RMS_level,lavfi.astats.2.RMS_level -of csv=p=0




*/
