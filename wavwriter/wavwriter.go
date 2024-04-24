// author: jialinwu

// Package wavwriter provides a media writer that writes bytes in μ-law (PCMU) or A-law (PCMA) format to a WAV file.
package wavwriter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/pion/rtp"
	"github.com/zaf/g711"
)

type WavWriter struct {
	sampleRate   uint32
	channelCount uint16
	pcmType      int

	fd      *os.File
	encoder *wav.Encoder
}

const (
	WavAudioFormatPcmRaw = 1 //no compression, raw (linear) pcm
	// a WavAudioFormat value of 6 or 7 would indicate that the audio data is stored in the A-law or μ-law format, which are forms of compressed PCM commonly used in telephony.
	WavAudioFormatPcmA = 6 //A-law
	WavAudioFormatPcmU = 7 //μ-law
)

var AvailablePcmTypes = []int{WavAudioFormatPcmRaw, WavAudioFormatPcmA, WavAudioFormatPcmU}

var ErrorInvalidPcmType = fmt.Errorf("invalid pcm type")

func New(fileName string, sampleRate uint32, channelCount uint16, pcmType int) (*WavWriter, error) {
	for i := 0; i < len(AvailablePcmTypes); i++ {
		if pcmType == AvailablePcmTypes[i] {
			break
		}
		if i == len(AvailablePcmTypes)-1 {
			return nil, ErrorInvalidPcmType
		}
	}

	f, err := os.Create(fileName) //nolint:gosec
	if err != nil {
		return nil, err
	}

	writer := &WavWriter{
		sampleRate:   sampleRate,
		channelCount: channelCount,
		pcmType:      pcmType,
	}
	writer.fd = f
	//BitDepth must be 16 bit there: while the encoded PCMA/PCMU data has a bit depth of 8 bits, the decoded audio data typically has a bit depth of 16 bits.
	//will convert to pcm raw before writting to file
	writer.encoder = wav.NewEncoder(f, int(sampleRate), 16, int(channelCount), WavAudioFormatPcmRaw)
	return writer, nil
}

func bytesToInt16ByReader(oneReader io.Reader) []int16 {
	data := make([]int16, 0, 512)
	for {
		var sample int16
		err := binary.Read(oneReader, binary.LittleEndian, &sample)
		switch {
		case err == io.EOF:
			return data
		case err != nil:
			return nil
		}
		data = append(data, sample)
	}
}

func bytesToInt16(buf []byte) []int16 {
	oneReader := bytes.NewReader(buf)
	return bytesToInt16ByReader(oneReader)
}

func (i *WavWriter) DecodePcmuToPcmWrite(pcmu []byte) error {
	b := bytes.NewBuffer(pcmu)
	udec, err := g711.NewUlawDecoder(b)
	if err != nil {
		panic(err.Error())
	}

	int16Data := bytesToInt16ByReader(udec)
	return i.WriteInt16(int16Data)
}

func (i *WavWriter) DecodePcmaToPcmWrite(pcma []byte) error {
	b := bytes.NewBuffer(pcma)
	udec, err := g711.NewAlawDecoder(b)
	if err != nil {
		panic(err.Error())
	}

	int16Data := bytesToInt16ByReader(udec)
	return i.WriteInt16(int16Data)
}

func (i *WavWriter) WriteInt16(int16Data []int16) error {
	intData := make([]int, len(int16Data))
	for i, v := range int16Data {
		intData[i] = int(v)
	}

	if err := i.encoder.Write(&audio.IntBuffer{Data: intData, Format: &audio.Format{SampleRate: int(i.sampleRate), NumChannels: int(i.channelCount)}}); err != nil {
		return err
	}
	return nil
}

func (i *WavWriter) Write(bs []byte) error {
	switch i.pcmType {
	case WavAudioFormatPcmRaw:
		int16Data := bytesToInt16(bs)
		return i.WriteInt16(int16Data)
	case WavAudioFormatPcmU:
		return i.DecodePcmuToPcmWrite(bs)
	case WavAudioFormatPcmA:
		return i.DecodePcmaToPcmWrite(bs)
	default:
		return fmt.Errorf("invalid pcm type: %d", i.pcmType)
	}
}

// WriteRTP adds a new packet and writes the appropriate headers for it
func (i *WavWriter) WriteRTP(packet *rtp.Packet) error {
	// decode Payload to PCM and Write the PCM data to the WAV file
	return i.Write(packet.Payload)
}

// Close stops the recording
func (i *WavWriter) Close() error {
	defer func() {
		i.fd = nil
		i.encoder = nil
	}()

	if err := i.encoder.Close(); err != nil {
		return err
	}

	if i.fd != nil {
		i.fd.Sync()
		return i.fd.Close()
	}
	return nil
}
