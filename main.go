//go:build !js
// +build !js

package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"time"
	"webrtcdemo/wavwriter"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/linnv/logx"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v4"
)

const (
	audioFileName     = "output.ogg"
	videoFileName     = "output.h264"
	oggPageDuration   = time.Millisecond * 20
	h264FrameDuration = time.Millisecond * 33
)

// Decode decodes the input from a JSON-encoded []byte.
func Decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func Mid() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		defer func() {
			if r := recover(); r != nil {
				c.AbortWithStatus(500)
				println("[ERROR] A panic has occurred:", r)
				debug.PrintStack()
			}
		}()
		c.Next()
	}
}

// HttpServer implements ...
func main() {
	dir := flag.String("d", "./jsfiddle", "dir to serve")
	flag.Parse()
	if !flag.Parsed() {
		os.Stderr.Write([]byte("ERROR: logging before flag.Parse"))
		return
	}

	port := ":8013"
	routers := gin.Default()
	// curl xxx:9081/
	routers.Use(static.Serve("/", static.LocalFile(*dir, true)))
	routers.Use(Mid())
	routers.Any("/offer", func(c *gin.Context) {

		// Wait for the offer to be pasted
		offer := webrtc.SessionDescription{}
		// rtcsignal.Decode(rtcsignal.MustReadStdin(), &offer)

		// Set the remote SessionDescription

		if err := Decode(c.Request.Body, &offer); err != nil {
			panic(err)
		}

		peerConnection := NewRtcConn()
		logx.Debugf("got client offer.SDP: %+v\n", offer.SDP)
		if err := peerConnection.SetRemoteDescription(offer); err != nil {
			logx.Warnf("err: %+v\n", err)
			panic(err)
		}

		// Create answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}

		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

		// Sets the LocalDescription, and starts our UDP listeners
		if err = peerConnection.SetLocalDescription(answer); err != nil {
			logx.Errorf("err: %+v\n", err)
			panic(err)
		}

		// Block until ICE Gathering is complete, disabling trickle ICE
		// we do this because we only can exchange one signaling message
		// in a production application you should exchange ICE Candidates via OnICECandidate
		<-gatherComplete
		oneSdp, _ := peerConnection.LocalDescription().Unmarshal()
		logx.Debugf("server gen oneSdp: %+v\n", oneSdp)
		// bs, err := json.MarshalIndent(peerConnection.LocalDescription(), "", "\t")
		// if err != nil {
		// 	logx.Errorf("err: %+v\n", err)
		// }
		// logx.Debugf("server answer.bs: %s\n", bs)
		// Output the answer in base64 so we can paste it in browser
		// fmt.Println(rtcsignal.Encode(*peerConnection.LocalDescription()))
		c.JSON(200, peerConnection.LocalDescription())
		// c.JSON(200, gin.H{
		// 	"message": "pong",
		// })
	})

	// curl xxx:9081/a
	// routers.Use(static.Serve("/a", static.LocalFile(*dir, true)))

	server := http.Server{
		Addr:    port,
		Handler: routers,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
			} else {
				panic(err)
			}
		}
	}()

	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	log.Print("http server listen on port ", port)
	log.Print("use c-c to exit: \n")
	<-sigChan
	server.Shutdown(nil)

}

func NewRtcConn() *webrtc.PeerConnection { //nolint
	// Assert that we have an audio or video file
	_, err := os.Stat(videoFileName)
	haveVideoFile := !os.IsNotExist(err)

	_, err = os.Stat(audioFileName)
	haveAudioFile := !os.IsNotExist(err)

	if !haveAudioFile && !haveVideoFile {
		panic("Could not find `" + audioFileName + "` or `" + videoFileName + "`")
	}

	// Create a MediaEngine object to configure the supported codec
	m := &webrtc.MediaEngine{}

	// // Setup the codecs you want to use.
	// // Only support VP8 and OPUS, this makes our WebM muxer code simpler
	// if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	// 	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	// 	PayloadType:        96,
	// }, webrtc.RTPCodecTypeVideo); err != nil {
	// 	panic(err)
	// }
	// if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	// 	// RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	// 	// PayloadType:        111,
	// }, webrtc.RTPCodecTypeAudio); err != nil {
	// 	panic(err)
	// }

	// if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	// 	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	// 	PayloadType:        8,
	// }, webrtc.RTPCodecTypeAudio); err != nil {
	// 	panic(err)
	// }

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        0,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	// if err := m.RegisterDefaultCodecs(); err != nil {
	// 	panic(err.Error())
	// }

	i := &interceptor.Registry{}

	// Register a intervalpli factory
	// This interceptor sends a PLI every 3 seconds. A PLI causes a video keyframe to be generated by the sender.
	// This makes our video seekable and more error resilent, but at a cost of lower picture quality and higher bitrates
	// A real world application should process incoming RTCP packets from viewers and forward them to senders
	intervalPliFactory, err := intervalpli.NewReceiverInterceptor()
	if err != nil {
		panic(err)
	}
	i.Add(intervalPliFactory)

	// Use the default set of Interceptors
	if err = webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}

	// Create the API object with the MediaEngine
	// api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))

	// Create a new RTCPeerConnection
	// peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				// URLs: []string{"stun:stun.l.google.com:19302"},

				// URLs: []string{"stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302", "stun:stun.l.google.com:19302", "stun:stun3.l.google.com:19302", "stun:stun4.l.google.com:19302"},

				URLs:       []string{"turn:192.168.1.8:3478"},
				Username:   "foo",
				Credential: "bar",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// pc.addTransceiver('audio', {'direction': 'sendrecv'})
	// no need to specified AddTransceiverFromKind will also got audio stream transceiver
	// if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
	// 	panic(err)
	// }

	// defer func() {
	// 	if cErr := peerConnection.Close(); cErr != nil {
	// 		fmt.Printf("cannot close peerConnection: %v\n", cErr)
	// 	}
	// }()

	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	isExist := false

	// oneFile, err := os.OpenFile("./server-got.pcm", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	log.Fatalf("failed opening file: %s", err)
	// }

	var oneWavWriter media.Writer
	oneWavWriter, err = wavwriter.New("./server-got.wav", 8000, 1, wavwriter.WavAudioFormatPcmU)
	if err != nil {
		panic(err.Error())
	}
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if errSend != nil {
					logx.Errorf("errSend: %+v\n", errSend)
					return
				}
			}
		}()

		logx.Debugfln("Track has started, of type %d: %s  codec %v\n", track.PayloadType(), track.Codec().RTPCodecCapability.MimeType, track.Codec())
		for {
			// Read RTP packets being sent to Pion
			rtp, _, readErr := track.ReadRTP()
			if readErr != nil {
				logx.Errorf("err: %+v\n", err)
				if readErr == io.EOF {
					// oneFile.Close()
					oneWavWriter.Close()
					return
				}
				panic(readErr)
			}
			switch track.Kind() {
			case webrtc.RTPCodecTypeAudio:
				logx.Debugf("rtp.String(): %+v\n", rtp.String())
				oneWavWriter.WriteRTP(rtp)
				// oneFile.Write(rtp.Payload)
				if isExist {
					// oneFile.Close()
					oneWavWriter.Close()
					return
				}
				// saver.PushOpus(rtp)
				// case webrtc.RTPCodecTypeVideo:
				// 	saver.PushVP8(rtp)
			}
		}
	})

	if haveAudioFile {
		go func() {
			<-iceConnectedCtx.Done()
			logx.Debugfln("iceConnectedCtx: client connected")
		}()
		// // Create a audio track
		// audioTrack, audioTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
		// if audioTrackErr != nil {
		// 	panic(audioTrackErr)
		// }
		//
		// rtpSender, audioTrackErr := peerConnection.AddTrack(audioTrack)
		// if audioTrackErr != nil {
		// 	panic(audioTrackErr)
		// }
		//
		// // Read incoming RTCP packets
		// // Before these packets are returned they are processed by interceptors. For things
		// // like NACK this needs to be called.
		// go func() {
		// 	rtcpBuf := make([]byte, 1500)
		// 	// ioutil.WriteFile("rtcp.log", []byte{}, 0644)
		// 	// Open the file in append mode. Create the file if it doesn't exist.
		// 	// oneFile, err := os.OpenFile("./server-got.pcm", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		// 	// if err != nil {
		// 	// 	log.Fatalf("failed opening file: %s", err)
		// 	// }
		// 	// defer oneFile.Close()
		// 	for {
		// 		if n, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
		// 			logx.Warnf("Error reading RTCP: %+v\n", rtcpErr)
		// 			return
		// 		} else {
		// 			// oneFile.Write(rtcpBuf[:n])
		// 			if isExist {
		// 				logx.Debugf("exit now: %+v\n", 1)
		// 				return
		// 			}
		// 			logx.Debugf("got len: %+v stream\n", n)
		// 		}
		// 	}
		// }()
		//
		// go func() {
		// 	// Open a ogg file and start reading using our oggReader
		// 	file, oggErr := os.Open(audioFileName)
		// 	if oggErr != nil {
		// 		panic(oggErr)
		// 	}
		//
		// 	// Open on oggfile in non-checksum mode.
		// 	ogg, _, oggErr := oggreader.NewWith(file)
		// 	if oggErr != nil {
		// 		panic(oggErr)
		// 	}
		//
		// 	// Wait for connection established
		// 	<-iceConnectedCtx.Done()
		//
		// 	// Keep track of last granule, the difference is the amount of samples in the buffer
		// 	var lastGranule uint64
		//
		// 	// It is important to use a time.Ticker instead of time.Sleep because
		// 	// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
		// 	// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
		// 	ticker := time.NewTicker(oggPageDuration)
		// 	for ; true; <-ticker.C {
		// 		pageData, pageHeader, oggErr := ogg.ParseNextPage()
		// 		if oggErr == io.EOF {
		// 			logx.Debugfln("All audio pages parsed and sent")
		// 			return
		// 			// os.Exit(0)
		// 		}
		//
		// 		if oggErr != nil {
		// 			panic(oggErr)
		// 		}
		//
		// 		// The amount of samples is the difference between the last and current timestamp
		// 		sampleCount := float64(pageHeader.GranulePosition - lastGranule)
		// 		lastGranule = pageHeader.GranulePosition
		// 		sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond
		//
		// 		if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
		// 			panic(oggErr)
		// 		}
		// 	}
		// }()
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		logx.Debugf("Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			iceConnectedCtxCancel()
		}
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		logx.Debugf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			logx.Debugf("Peer Connection has gone to failed exiting")
			logx.Flush()
			os.Exit(0)
		}
	})
	return peerConnection
}
