package main

import (
	"fmt"
	"log"
	"main/internal/webrtcserver"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

func createPionRTPPacket(p *discordgo.Packet) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			Version: 2,
			// Taken from Discord voice docs
			PayloadType:    0x78,
			SequenceNumber: p.Sequence,
			Timestamp:      p.Timestamp,
			SSRC:           p.SSRC,
		},
		Payload: p.Opus,
	}
}

func handleVoice(c chan *discordgo.Packet, track *webrtc.TrackLocalStaticRTP) {

	files := make(map[uint32]media.Writer)
	for p := range c {
		file, ok := files[p.SSRC]
		if !ok {
			log.Println("create new writer and file")
			var err error
			file, err = oggwriter.New(fmt.Sprintf("./recordings/%d.ogg", p.SSRC), 48000, 2)
			if err != nil {
				fmt.Printf("failed to create file %d.ogg, giving up on recording: %v\n", p.SSRC, err)
				return
			}
			files[p.SSRC] = file
		}
		// Construct pion RTP packet from DiscordGo's type.
		rtp := createPionRTPPacket(p)
		err := file.WriteRTP(rtp)
		if err != nil {
			fmt.Printf("failed to write to file %d.ogg, giving up on recording: %v\n", p.SSRC, err)
		}

		//aaa

		err = track.WriteRTP(rtp)
		if err != nil {
			log.Fatal(err)
		}

	}

	// Once we made it here, we're done listening for packets. Close all files
	for _, f := range files {
		f.Close()
	}
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

	// We only really care about receiving voice state updates.
	s.Identify.Intents = discordgo.IntentsGuildVoiceStates

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
	var channel *discordgo.Channel
	for _, n := range channels {
		if n.Type == discordgo.ChannelTypeGuildVoice {
			channel = n
			break
		}
	}
	if channel == nil {
		log.Fatal("no voice channel")
	}
	ChannelID := channel.ID

	fmt.Println(ChannelID)
	log.Println("RUNN")
	track := webrtcserver.Run()

	v, err := s.ChannelVoiceJoin(GuildID, ChannelID, true, false)
	if err != nil {
		log.Fatal("failed to join voice channel:", err)
		return
	}

	go func() {
		log.Println("recording a lot of  sec")
		time.Sleep(10000 * time.Second)
		close(v.OpusRecv)
		log.Println("stopped recording")
		v.Close()
	}()

	handleVoice(v.OpusRecv, track)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}
