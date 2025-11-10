//go:build !js
// +build !js

package webrtcserver

import (
	"fmt"
	"main/internal/signal"

	"github.com/pion/webrtc/v3"
)

func Run() *webrtc.TrackLocalStaticRTP {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Create an audio track
	fmt.Println("Creating audio track with Opus codec...")
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if err != nil {
		panic(err)
	}
	fmt.Println("Audio track created successfully")
	
	rtpSender, err := peerConnection.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}
	fmt.Println("Audio track added to peer connection")

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())

		if connectionState == webrtc.ICEConnectionStateConnected || connectionState == webrtc.ICEConnectionStateCompleted {
			fmt.Println("✅ WebRTC connection established! Audio should start streaming...")
		}

		if connectionState == webrtc.ICEConnectionStateFailed {
			fmt.Println("❌ WebRTC connection failed!")
			if closeErr := peerConnection.Close(); closeErr != nil {
				panic(closeErr)
			}
		}
	})
	
	// Add track event handler to see when browser receives the track
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Printf("Browser received track: %s, kind: %s\n", track.ID(), track.Kind())
	})

	fmt.Println("Waiting for offer in 'offer.txt' file...")
	fmt.Println("(Paste your base64 offer into offer.txt and save the file)")

	// Wait for the offer to be written to file
	offer := webrtc.SessionDescription{}
	offerStr := signal.MustReadFromFile("offer.txt")
	fmt.Println("Received offer, length:", len(offerStr))
	signal.Decode(offerStr, &offer)

	// Clear the file after reading
	signal.ClearFile("offer.txt")

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
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
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(signal.Encode(*peerConnection.LocalDescription()))

	return audioTrack

}
