module github.com/chdxd1/nanoleaf-artnet

go 1.17

require (
	github.com/adnanbrq/nanoleaf v0.0.0-20201213184944-913d5ef05e03
	github.com/jsimonetti/go-artnet v0.0.0-20210922080205-810e8e5e57a2
)

require (
	github.com/go-resty/resty/v2 v2.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/net v0.0.0-20200222125558-5a598a2470a0 // indirect
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
)

replace github.com/jsimonetti/go-artnet => github.com/chdxD1/go-artnet v0.0.0-20211220220028-30ac920ae1d9

replace github.com/adnanbrq/nanoleaf => github.com/chdxD1/nanoleaf-go v0.0.0-20211220204931-24fad2cbe599
