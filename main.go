package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/adnanbrq/nanoleaf"
	"github.com/jsimonetti/go-artnet"
	"github.com/jsimonetti/go-artnet/packet"
	"github.com/jsimonetti/go-artnet/packet/code"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type nanoleafConfig struct {
	Api   string `mapstructure:"api"`
	Token string `mapstructure:"token"`
}

type artnetConfig struct {
	InterfaceName string `mapstructure:"interfaceName"`
	StartAddress  uint16 `mapstructure:"startAddress"`
}

type config struct {
	Nanoleafs []nanoleafConfig `mapstructure:"nanoleafs"`
	ArtNet    artnetConfig     `mapstructure:"artnet"`
}

type ArtNetNanoleaf struct {
	nanoleaf             *nanoleaf.Nanoleaf
	startUniverseAddress uint16
	consumedUniverses    uint16
	positionData         []nanoleaf.PanelPositionData
}

var (
	DMX_CHANNELS      = 512
	CHANNEL_PER_PANEL = 4
	PANEL_IN_UNIVERSE = DMX_CHANNELS / CHANNEL_PER_PANEL
)

func waitForAllTokens(nlfs []*nanoleaf.Nanoleaf) {
	// First identify/flash all Nanoleafs where we have a token, remove token if Unauthorized
	for _, nlf := range nlfs {
		if nlf.Identity.Flash() == nanoleaf.ErrUnauthorized {
			nlf.SetToken("")
		}
	}

	for {
		waitingAuth := 0
		for _, nlf := range nlfs {
			if nlf.IsConnected() {
				continue
			}
			fmt.Printf("Trying to authenticate nanoleaf at %s\n", nlf.GetUrl())
			if nlf.Auth.Authenticate() != nil {
				fmt.Printf("nanoleaf at %s not yet ready - please set to pairing mode according to guide\n", nlf.GetUrl())
				waitingAuth++
			} else {
				fmt.Printf("nanoleaf at %s successfully authenticated", nlf.GetUrl())
			}

		}
		if waitingAuth == 0 {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func main() {
	log := artnet.NewLogger(logrus.New().WithFields(nil))

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("fatal error config file: %w", err)
	}

	var C config

	viper.Unmarshal(&C)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	// If no Nanoleafs were configured we will first discover them through mDNS
	if len(C.Nanoleafs) == 0 {
		nanoleafs, err := nanoleaf.DiscoverNanoleafs(5 * time.Second)
		if err != nil {
			log.Fatalf("couldn't discover Nanoleafs, %v", err)
		}
		for _, discoveredNanoleafs := range nanoleafs {
			C.Nanoleafs = append(C.Nanoleafs, nanoleafConfig{
				Api: discoveredNanoleafs.GetUrl(),
			})
		}
		viper.Set("nanoleafs", C.Nanoleafs)
		viper.WriteConfig()
	}

	nanoleafs := []*nanoleaf.Nanoleaf{}
	for _, nanoleafAPI := range C.Nanoleafs {
		nlf := nanoleaf.NewNanoleaf(nanoleafAPI.Api)
		if len(nanoleafAPI.Token) > 0 {
			nlf.SetToken(nanoleafAPI.Token)
		}
		nanoleafs = append(nanoleafs, nlf)
	}

	// If some Nanoleafs do not have a token set, try to login for some time
	waitForAllTokens(nanoleafs)

	artNetNanoleafs := []ArtNetNanoleaf{}
	universeOffset := uint16(0)

	for _, nlfs := range nanoleafs {
		err = nlfs.Stream.Activate(nanoleaf.VersionV2) // TODO: Autodetect version 1 / version 2
		if err != nil {
			panic(err)
		}

		layout, _ := nlfs.Layout.GetLayout()
		positions := layout.PositionData
		sort.Slice(positions, func(i, j int) bool {
			if positions[i].Y == positions[j].Y {
				return positions[i].X < positions[j].X
			} else {
				return positions[i].Y > positions[j].Y
			}
		})

		err = nlfs.Stream.Connect()
		if err != nil {
			panic(err)
		}

		neededDMXAddresses := len(positions) * CHANNEL_PER_PANEL
		neededUniverses := uint16((neededDMXAddresses + DMX_CHANNELS - 1) / DMX_CHANNELS)

		artNetNanoleafs = append(artNetNanoleafs, ArtNetNanoleaf{
			nanoleaf:             nlfs,
			startUniverseAddress: C.ArtNet.StartAddress + universeOffset,
			consumedUniverses:    neededUniverses,
			positionData:         positions,
		})
		universeOffset += neededUniverses
	}

	// Initialize Art-Net node
	node, err := artnet.NewNode("nanoleaf", code.StNode, C.ArtNet.InterfaceName, log)
	if err != nil {
		panic(err)
	}

	// Add ports for universe
	for universe := uint16(0); universe < universeOffset; universe++ {
		node.Config.OutputPorts = append(node.Config.OutputPorts, artnet.OutputPort{
			Address: artnet.AddressFromInt(universe),
			Type:    code.PortType(0).WithType("DMX512"),
		})
	}

	// Calback for DMX data packet
	node.RegisterCallback(code.OpDMX, func(p packet.ArtNetPacket) {
		dmxPacket := p.(*packet.ArtDMXPacket)

		artnetAddress := artnet.Address{
			Net:    dmxPacket.Net,
			SubUni: dmxPacket.SubUni,
		}
		address := uint16(artnetAddress.Integer())

		// Find initialized nanoleaf by universe
		var artNetNanoleaf *ArtNetNanoleaf
		offset := 0
		for _, annlf := range artNetNanoleafs {
			endUniverse := annlf.startUniverseAddress + annlf.consumedUniverses - 1
			if annlf.startUniverseAddress <= address && address <= endUniverse {
				artNetNanoleaf = &annlf
				offset = int(address - annlf.startUniverseAddress)
				break
			}
		}
		if artNetNanoleaf == nil {
			fmt.Printf("Did not find corresponding Nanoleaf for received ArtDMXPacket in universe %s\n", artnetAddress.String())
			return
		}

		panels := []nanoleaf.PanelEffect{}

		// Calculate panels to update
		positionStart := offset * PANEL_IN_UNIVERSE
		positionEnd := min((offset+1)*PANEL_IN_UNIVERSE, len(artNetNanoleaf.positionData))

		for i, panel := range artNetNanoleaf.positionData[positionStart:positionEnd] {
			panels = append(panels, nanoleaf.PanelEffect{
				ID: panel.ID,
				Frame: nanoleaf.FrameEffect{
					Red:   int(dmxPacket.Data[i*CHANNEL_PER_PANEL+0]),
					Green: int(dmxPacket.Data[i*CHANNEL_PER_PANEL+1]),
					Blue:  int(dmxPacket.Data[i*CHANNEL_PER_PANEL+2]),
					White: int(dmxPacket.Data[i*CHANNEL_PER_PANEL+3]),
				},
			})
		}

		artNetNanoleaf.nanoleaf.Stream.WriteEffect(nanoleaf.StreamEffect{
			Panels: panels,
		})
	})

	defer node.Stop()
	node.Start()

	for {
		time.Sleep(10 * time.Second)
	}
}
