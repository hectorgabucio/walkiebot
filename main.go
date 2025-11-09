package main

import (
	"fmt"
	"io"
	"log"
	"main/internal/webrtcserver"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

// RTPStreamState maintains continuous sequence numbers and timestamps across multiple voice messages
type RTPStreamState struct {
	mu             sync.Mutex
	sequenceNumber uint16
	timestamp      uint32
}

// Extract Opus packets from OGG container
// OGG pages contain segments, and segments contain Opus packets
func extractOpusPackets(data []byte) ([][]byte, error) {
	var opusPackets [][]byte
	pos := 0

	for pos < len(data) {
		// OGG page starts with "OggS"
		if pos+4 > len(data) || string(data[pos:pos+4]) != "OggS" {
			break
		}

		// Read page header (27 bytes minimum)
		if pos+27 > len(data) {
			break
		}

		// Get page segment count (byte 26)
		segmentCount := int(data[pos+26])
		if pos+27+segmentCount > len(data) {
			break
		}

		// Read segment table to get segment sizes
		pageHeaderSize := 27 + segmentCount
		segmentSizes := make([]int, segmentCount)
		totalSegmentsSize := 0
		for i := 0; i < segmentCount; i++ {
			segmentSizes[i] = int(data[pos+27+i])
			totalSegmentsSize += segmentSizes[i]
		}

		pageSize := pageHeaderSize + totalSegmentsSize
		if pos+pageSize > len(data) {
			break
		}

		// Extract segments (which contain Opus packets)
		segmentStart := pos + pageHeaderSize
		for _, segSize := range segmentSizes {
			if segSize > 0 && segmentStart+segSize <= len(data) {
				// Each segment is an Opus packet
				opusPacket := make([]byte, segSize)
				copy(opusPacket, data[segmentStart:segmentStart+segSize])
				if len(opusPacket) > 0 {
					opusPackets = append(opusPackets, opusPacket)
				}
				segmentStart += segSize
			}
		}

		pos += pageSize
	}

	return opusPackets, nil
}

// Stream OGG/Opus audio file to WebRTC track with continuous sequence numbers
func streamVoiceMessage(audioData []byte, track *webrtc.TrackLocalStaticRTP, rtpState *RTPStreamState) error {
	log.Println("Streaming voice message to WebRTC...")

	// Extract Opus packets from OGG container
	opusPackets, err := extractOpusPackets(audioData)
	if err != nil {
		return fmt.Errorf("failed to extract Opus packets: %v", err)
	}

	if len(opusPackets) == 0 {
		return fmt.Errorf("no Opus packets found in audio data")
	}

	log.Printf("Found %d Opus packets, streaming...\n", len(opusPackets))

	packetCount := 0

	// Lock to get current sequence number and timestamp
	rtpState.mu.Lock()
	currentSeq := rtpState.sequenceNumber
	currentTs := rtpState.timestamp
	rtpState.mu.Unlock()

	for _, opusPacket := range opusPackets {
		if len(opusPacket) == 0 {
			continue
		}

		// Create RTP packet with Opus payload
		rtpPacket := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    111, // Opus payload type
				SequenceNumber: currentSeq,
				Timestamp:      currentTs,
				SSRC:           123456789, // Fixed SSRC for voice messages
			},
			Payload: opusPacket,
		}

		// Write to WebRTC track
		if err := track.WriteRTP(rtpPacket); err != nil {
			return fmt.Errorf("error writing RTP packet: %v", err)
		}

		currentSeq++
		// Timestamp increments by samples per packet (typically 960 for 20ms Opus frames at 48kHz)
		currentTs += 960
		packetCount++

		if packetCount == 1 {
			log.Println("✅ Successfully streamed first packet to WebRTC!")
		}

		// Small delay to simulate real-time streaming (20ms per packet)
		time.Sleep(20 * time.Millisecond)
	}

	// Update global state with new values
	rtpState.mu.Lock()
	rtpState.sequenceNumber = currentSeq
	rtpState.timestamp = currentTs
	rtpState.mu.Unlock()

	log.Printf("✅ Finished streaming voice message. Total packets: %d (next seq: %d, next ts: %d)\n",
		packetCount, currentSeq, currentTs)
	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	Token := os.Getenv("DISCORD_TOKEN")

	s, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal("error creating Discord session:", err)
	}
	defer s.Close()

	// We need to receive messages to detect voice messages
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	err = s.Open()
	if err != nil {
		log.Fatal("error opening connection:", err)
	}

	guilds, err := s.UserGuilds(0, "", "")
	if err != nil {
		log.Fatal("could not get guilds")
	}
	guild := guilds[0]
	GuildID := guild.ID
	channels, err := s.GuildChannels(GuildID)
	if err != nil {
		log.Fatal("could not get channels")
	}

	// Find the first text channel instead of voice channel
	var textChannel *discordgo.Channel
	for _, n := range channels {
		if n.Type == discordgo.ChannelTypeGuildText {
			textChannel = n
			break
		}
	}
	if textChannel == nil {
		log.Fatal("no text channel found")
	}

	log.Printf("✅ Found text channel: %s (%s)\n", textChannel.Name, textChannel.ID)
	log.Println("🎤 Bot will listen for voice messages in this channel!")

	// Initialize WebRTC
	track := webrtcserver.Run()

	// Create RTP stream state to maintain continuous sequence numbers across messages
	rtpState := &RTPStreamState{
		sequenceNumber: 0,
		timestamp:      0,
	}

	// Message handler for voice messages
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages from bots
		if m.Author.Bot {
			return
		}

		// Only process messages in the target channel
		if m.ChannelID != textChannel.ID {
			return
		}

		// Check for voice message attachments
		// Voice messages typically have .ogg extension or are flagged as voice messages
		for _, attachment := range m.Attachments {
			isVoiceMessage := false

			// Check if it's a voice message by extension
			if strings.HasSuffix(strings.ToLower(attachment.Filename), ".ogg") ||
				strings.HasSuffix(strings.ToLower(attachment.Filename), ".opus") {
				isVoiceMessage = true
			}

			// Also check if Discord flags it as a voice message (usually in the filename)
			if strings.Contains(strings.ToLower(attachment.Filename), "voice-message") {
				isVoiceMessage = true
			}

			if isVoiceMessage {
				log.Printf("🎤 Voice message detected from %s! Downloading...\n", m.Author.Username)

				// Download the voice message
				resp, err := http.Get(attachment.URL)
				if err != nil {
					log.Printf("❌ Error downloading voice message: %v\n", err)
					continue
				}
				defer resp.Body.Close()

				audioData, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("❌ Error reading voice message: %v\n", err)
					continue
				}

				log.Printf("✅ Downloaded voice message (%d bytes), streaming to WebRTC...\n", len(audioData))

				// Stream to WebRTC with continuous sequence numbers
				if err := streamVoiceMessage(audioData, track, rtpState); err != nil {
					log.Printf("❌ Error streaming voice message: %v\n", err)
				} else {
					log.Println("✅ Voice message streamed successfully!")
				}
			}
		}
	})

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}
