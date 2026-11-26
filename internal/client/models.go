package client

type VLAN struct {
	ID   uint16
	Name string
}

type EthernetInterface struct {
	Name         string
	Description  string
	Enabled      bool
	AccessVlan   uint16
	AllowedVlans []uint16
}
type PortMode string

const (
	PortModeAccess = "access"
	PortModeTrunk  = "trunk"
)
