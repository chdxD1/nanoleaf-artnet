package main

import (
	"net"
	"os"
	"sort"
	"time"

	"github.com/adnanbrq/nanoleaf"
	"github.com/jsimonetti/go-artnet"
	"github.com/jsimonetti/go-artnet/packet"
	"github.com/jsimonetti/go-artnet/packet/code"
)

func main() {
	nlfs := nanoleaf.NewNanoleaf(os.Getenv("NANOLEAF_URL"))
	nlfs.SetToken(os.Getenv("NANOLEAF_TOKEN"))

	err := nlfs.Stream.Activate(nanoleaf.VersionV2)
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

	log := artnet.NewDefaultLogger()
	node := artnet.NewNode("nanoleaf", code.StNode, net.ParseIP(os.Getenv("ARTNET_NODE_IP")), log)

	node.RegisterCallback(code.OpDMX, func(p packet.ArtNetPacket) {
		dmxPacket := p.(*packet.ArtDMXPacket)

		panels := []nanoleaf.PanelEffect{}

		for i, panel := range positions {
			panels = append(panels, nanoleaf.PanelEffect{
				ID: panel.ID,
				Frame: nanoleaf.FrameEffect{
					Red:   int(dmxPacket.Data[i*3]),
					Green: int(dmxPacket.Data[i*3+1]),
					Blue:  int(dmxPacket.Data[i*3+2]),
				},
			})
		}

		nlfs.Stream.WriteEffect(nanoleaf.StreamEffect{
			Panels: panels,
		})
	})

	defer node.Stop()
	node.Start()

	for {
		time.Sleep(10 * time.Second)
	}
}
