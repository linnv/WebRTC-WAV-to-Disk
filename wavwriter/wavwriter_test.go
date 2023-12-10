package wavwriter

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/go-audio/wav"
	"github.com/linnv/logx"
)

type DiscardWriteSeeker struct{}

func (ws *DiscardWriteSeeker) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (ws *DiscardWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func BenchmarkWavWriter_Write(b *testing.B) {
	pcmuBs, err := ioutil.ReadFile("/Users/jialinwu/qn-pc/tmp/server-got-encoded.pcmu")
	var mockWriter = &DiscardWriteSeeker{}
	sampleRate := 8000
	channelCount := 1

	writer := &WavWriter{
		sampleRate:   uint32(sampleRate),
		channelCount: uint16(channelCount),
		// encoder:      wav.NewEncoder(mockWriter, int(sampleRate), 16, int(channelCount), WavAudioFormatPcmU),
		encoder: wav.NewEncoder(mockWriter, int(sampleRate), 16, int(channelCount), WavAudioFormatPcmU),
	}
	// writer.fd = f
	// Create a new encoder
	// a WavAudioFormat value of 6 or 7 would indicate that the audio data is stored in the A-law or μ-law format, which are forms of compressed PCM commonly used in telephony.
	//must be 16 bit there:while the encoded PCMA/PCMU data has a bit depth of 8 bits, the decoded audio data typically has a bit depth of 16 bits.
	// writer.encoder = wav.NewEncoder(f, int(sampleRate), 16, int(channelCount), WavAudioFormatPcmU)
	// f, err := os.Open("/Users/jialinwu/qn-pc/tmp/server-got-encoded.pcmu")
	if err != nil {
		panic(fmt.Sprintf("couldn't open audio file - %v", err))
	}
	for i := 0; i < b.N; i++ {
		// writer.Write(pcmuBs)
		writer.Write(pcmuBs)
	}
}

func TestWavWriter_Write(t *testing.T) {
	// /Users/jialinwu/qn-pc/tmp/server-got-encoded.pcmu
	pcmuBs, err := ioutil.ReadFile("/Users/jialinwu/qn-pc/tmp/server-got-encoded.pcmu")

	const (
		sampleRate   = 8000
		channelCount = 1
	)

	// writer, err := New("./server-got-pcmu.wav", sampleRate, channelCount)
	writer, err := New("./server-got-pcmubyg711.wav", sampleRate, channelCount, WavAudioFormatPcmU)
	// f, err := os.Open("/Users/jialinwu/qn-pc/tmp/server-got-encoded.pcmu")
	if err != nil {
		panic(fmt.Sprintf("couldn't open audio file - %v", err))
	}

	type fields struct {
		// sampleRate   uint32
		// channelCount uint16
		// stream       io.Writer
		// fd           *os.File
		encoder *wav.Encoder
	}
	type args struct {
		pcmu []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"normal", fields{nil}, args{pcmuBs}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// i := &WavWriter{
			// 	// sampleRate:   tt.fields.sampleRate,
			// 	// channelCount: tt.fields.channelCount,
			// 	// stream:       tt.fields.stream,
			// 	// fd:           tt.fields.fd,
			// 	encoder: tt.fields.encoder,
			// }
			// if err := writer.Write(pcmuBs); (err != nil) != tt.wantErr
			if err := writer.Write(pcmuBs); (err != nil) != tt.wantErr {
				t.Errorf("WavWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	writer.Close()
	logx.Flush()
}
