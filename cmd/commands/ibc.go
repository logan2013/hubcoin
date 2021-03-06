package commands

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/urfave/cli"

	"github.com/dragosroua/hubcoin/plugins/ibc"

	cmn "github.com/tendermint/go-common"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
	tmtypes "github.com/tendermint/tendermint/types"
)

// returns a new IBC plugin to be registered with Hubcoin
func NewIBCPlugin() *ibc.IBCPlugin {
	return ibc.New()
}

//---------------------------------------------------------------------
// ibc flags

var (
	IbcChainIDFlag = cli.StringFlag{
		Name:  "chain_id",
		Usage: "ChainID for the new blockchain",
		Value: "",
	}

	IbcGenesisFlag = cli.StringFlag{
		Name:  "genesis",
		Usage: "Genesis file for the new blockchain",
		Value: "",
	}

	IbcHeaderFlag = cli.StringFlag{
		Name:  "header",
		Usage: "Block header for an ibc update",
		Value: "",
	}

	IbcCommitFlag = cli.StringFlag{
		Name:  "commit",
		Usage: "Block commit for an ibc update",
		Value: "",
	}

	IbcFromFlag = cli.StringFlag{
		Name:  "from",
		Usage: "Source ChainID",
		Value: "",
	}

	IbcToFlag = cli.StringFlag{
		Name:  "to",
		Usage: "Destination ChainID",
		Value: "",
	}

	IbcTypeFlag = cli.StringFlag{
		Name:  "type",
		Usage: "IBC packet type (eg. coin)",
		Value: "",
	}

	IbcPayloadFlag = cli.StringFlag{
		Name:  "payload",
		Usage: "IBC packet payload",
		Value: "",
	}

	IbcPacketFlag = cli.StringFlag{
		Name:  "packet",
		Usage: "hex-encoded IBC packet",
		Value: "",
	}

	IbcProofFlag = cli.StringFlag{
		Name:  "proof",
		Usage: "hex-encoded proof of IBC packet from source chain",
		Value: "",
	}

	IbcSequenceFlag = cli.IntFlag{
		Name:  "sequence",
		Usage: "sequence number for IBC packet",
		Value: 0,
	}

	IbcHeightFlag = cli.IntFlag{
		Name:  "height",
		Usage: "Height the packet became egress in source chain",
		Value: 0,
	}
)

//---------------------------------------------------------------------
// ibc commands

var (
	IbcTxCmd = cli.Command{
		Name:  "ibc",
		Usage: "an IBC transaction, for InterBlockchain Communication",
		Flags: TxFlags,
		Subcommands: []cli.Command{
			IbcRegisterTxCmd,
			IbcUpdateTxCmd,
			IbcPacketTxCmd,
		},
	}

	IbcRegisterTxCmd = cli.Command{
		Name:  "register",
		Usage: "Register a blockchain via IBC",
		Action: func(c *cli.Context) error {
			return cmdIBCRegisterTx(c)
		},
		Flags: []cli.Flag{
			IbcChainIDFlag,
			IbcGenesisFlag,
		},
	}

	IbcUpdateTxCmd = cli.Command{
		Name:  "update",
		Usage: "Update the latest state of a blockchain via IBC",
		Action: func(c *cli.Context) error {
			return cmdIBCUpdateTx(c)
		},
		Flags: []cli.Flag{
			IbcHeaderFlag,
			IbcCommitFlag,
		},
	}

	IbcPacketTxCmd = cli.Command{
		Name:  "packet",
		Usage: "Send a new packet via IBC",
		Subcommands: []cli.Command{
			IbcPacketCreateTx,
			IbcPacketPostTx,
		},
	}

	IbcPacketCreateTx = cli.Command{
		Name:  "create",
		Usage: "Create an egress IBC packet",
		Action: func(c *cli.Context) error {
			return cmdIBCPacketCreateTx(c)
		},
		Flags: []cli.Flag{
			IbcFromFlag,
			IbcToFlag,
			IbcTypeFlag,
			IbcPayloadFlag,
			IbcSequenceFlag,
		},
	}

	IbcPacketPostTx = cli.Command{
		Name:  "post",
		Usage: "Deliver an IBC packet to another chain",
		Action: func(c *cli.Context) error {
			return cmdIBCPacketPostTx(c)
		},
		Flags: []cli.Flag{
			IbcFromFlag,
			IbcHeightFlag,
			IbcPacketFlag,
			IbcProofFlag,
		},
	}
)

//---------------------------------------------------------------------
// ibc command implementations

func cmdIBCRegisterTx(c *cli.Context) error {
	chainID := c.String("chain_id")
	genesisFile := c.String("genesis")
	parent := c.Parent()

	genesisBytes, err := ioutil.ReadFile(genesisFile)
	if err != nil {
		return errors.New(cmn.Fmt("Error reading genesis file %v: %v", genesisFile, err))
	}

	ibcTx := ibc.IBCRegisterChainTx{
		ibc.BlockchainGenesis{
			ChainID: chainID,
			Genesis: string(genesisBytes),
		},
	}

	fmt.Println("IBCTx:", string(wire.JSONBytes(ibcTx)))

	data := []byte(wire.BinaryBytes(struct {
		ibc.IBCTx `json:"unwrap"`
	}{ibcTx}))
	name := "IBC"

	return AppTx(parent, name, data)
}

func cmdIBCUpdateTx(c *cli.Context) error {
	headerBytes, err := hex.DecodeString(StripHex(c.String("header")))
	if err != nil {
		return errors.New(cmn.Fmt("Header (%v) is invalid hex: %v", c.String("header"), err))
	}
	commitBytes, err := hex.DecodeString(StripHex(c.String("commit")))
	if err != nil {
		return errors.New(cmn.Fmt("Commit (%v) is invalid hex: %v", c.String("commit"), err))
	}

	header := new(tmtypes.Header)
	commit := new(tmtypes.Commit)

	if err := wire.ReadBinaryBytes(headerBytes, &header); err != nil {
		return errors.New(cmn.Fmt("Error unmarshalling header: %v", err))
	}
	if err := wire.ReadBinaryBytes(commitBytes, &commit); err != nil {
		return errors.New(cmn.Fmt("Error unmarshalling commit: %v", err))
	}

	ibcTx := ibc.IBCUpdateChainTx{
		Header: *header,
		Commit: *commit,
	}

	fmt.Println("IBCTx:", string(wire.JSONBytes(ibcTx)))

	data := []byte(wire.BinaryBytes(struct {
		ibc.IBCTx `json:"unwrap"`
	}{ibcTx}))
	name := "IBC"

	return AppTx(c.Parent(), name, data)
}

func cmdIBCPacketCreateTx(c *cli.Context) error {
	fromChain, toChain := c.String("from"), c.String("to")
	packetType := c.String("type")

	payloadBytes, err := hex.DecodeString(StripHex(c.String("payload")))
	if err != nil {
		return errors.New(cmn.Fmt("Payload (%v) is invalid hex: %v", c.String("payload"), err))
	}

	sequence, err := getIBCSequence(c)
	if err != nil {
		return err
	}

	ibcTx := ibc.IBCPacketCreateTx{
		Packet: ibc.Packet{
			SrcChainID: fromChain,
			DstChainID: toChain,
			Sequence:   sequence,
			Type:       packetType,
			Payload:    payloadBytes,
		},
	}

	fmt.Println("IBCTx:", string(wire.JSONBytes(ibcTx)))

	data := []byte(wire.BinaryBytes(struct {
		ibc.IBCTx `json:"unwrap"`
	}{ibcTx}))

	return AppTx(c.Parent().Parent(), "IBC", data)
}

func cmdIBCPacketPostTx(c *cli.Context) error {
	fromChain, fromHeight := c.String("from"), c.Int("height")

	packetBytes, err := hex.DecodeString(StripHex(c.String("packet")))
	if err != nil {
		return errors.New(cmn.Fmt("Packet (%v) is invalid hex: %v", c.String("packet"), err))
	}
	proofBytes, err := hex.DecodeString(StripHex(c.String("proof")))
	if err != nil {
		return errors.New(cmn.Fmt("Proof (%v) is invalid hex: %v", c.String("proof"), err))
	}

	var packet ibc.Packet
	proof := new(merkle.IAVLProof)

	if err := wire.ReadBinaryBytes(packetBytes, &packet); err != nil {
		return errors.New(cmn.Fmt("Error unmarshalling packet: %v", err))
	}
	if err := wire.ReadBinaryBytes(proofBytes, &proof); err != nil {
		return errors.New(cmn.Fmt("Error unmarshalling proof: %v", err))
	}

	ibcTx := ibc.IBCPacketPostTx{
		FromChainID:     fromChain,
		FromChainHeight: uint64(fromHeight),
		Packet:          packet,
		Proof:           proof,
	}

	fmt.Println("IBCTx:", string(wire.JSONBytes(ibcTx)))

	data := []byte(wire.BinaryBytes(struct {
		ibc.IBCTx `json:"unwrap"`
	}{ibcTx}))

	return AppTx(c.Parent().Parent(), "IBC", data)
}

func getIBCSequence(c *cli.Context) (uint64, error) {
	if c.IsSet("sequence") {
		return uint64(c.Int("sequence")), nil
	}

	// TODO: get sequence
	return 0, nil
}
