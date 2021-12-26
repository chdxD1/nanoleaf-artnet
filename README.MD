# nanoleaf-artnet Bridge

This tool works as a bridge between Art-Net 4 from (c) Artistic Licence Holding Ltd and Nanoleaf modules. It is tested with Nanoleaf Canvas but should work with other Nanoleaf products as well, as long as the autodetection of the streaming version is implemented.

## config.yml
```yaml
artnet:
  startaddress: 0
  interfaceName: "Ethernet"
nanoleafs: []
```

The basic configuration has to exist as `config.yml` in the same folder as the executable. It needs to contain the above settings. It is required to set a start universe address for Art-Net and an interface name for the Art-Net communication. The interface name can easily be checked by running `ipconfig` on a Windows machine.

We announce ourselfs as `nanoleaf` into the Art-Net network with as many ports/universes as needed for the panels to fit in.

## Nanoleaf auto-configuration
When no nanoleafs are configured in the config we try to auto-discover them by using mDNS. After the discovery the tool will exit, giving you the possibility to configure any existing tokens you have for the installation.

After some nanoleafs were configured it checks if a token is set in the config. If a token is set we first try to identify the device by flashing. If no token is set we will try to authenticate to the Nanoleafs in a loop. Please refer to the Nanoleaf manual how to put the product into pairing/authentication mode. For Nanoleaf Canvas you need to hold down the power button for 10 seconds.

## Universe auto-configuration
Each nanoleaf installation, by the ordering in the config, will consume at least one full Art-Net universe. There will never be two Nanoleaf installations in the same universe. That said each panel will consume 4 addresses (R, G, B, W) however white might not be supported by your product. You will end up with 128 panels in each universe. If your installation has more panels another universe will be used. 