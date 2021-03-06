package app

import (
	"strings"

	abci "github.com/tendermint/abci/types"
	sm "github.com/dragosroua/hubcoin/state"
	"github.com/dragosroua/hubcoin/types"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	eyes "github.com/tendermint/merkleeyes/client"
)

const (
	version   = "0.1"
	maxTxSize = 10240

	PluginNameBase = "hub"
)

type Hubcoin struct {
	eyesCli    *eyes.Client
	state      *sm.State
	cacheState *sm.State
	plugins    *types.Plugins
}

func NewHubcoin(eyesCli *eyes.Client) *Hubcoin {
	state := sm.NewState(eyesCli)
	plugins := types.NewPlugins()
	return &Hubcoin{
		eyesCli:    eyesCli,
		state:      state,
		cacheState: nil,
		plugins:    plugins,
	}
}

// For testing, not thread safe!
func (app *Hubcoin) GetState() *sm.State {
	return app.state.CacheWrap()
}

// ABCI::Info
func (app *Hubcoin) Info() abci.ResponseInfo {
	return abci.ResponseInfo{Data: Fmt("Hubcoin v%v", version)}
}

func (app *Hubcoin) RegisterPlugin(plugin types.Plugin) {
	app.plugins.RegisterPlugin(plugin)
}

// ABCI::SetOption
func (app *Hubcoin) SetOption(key string, value string) string {
	pluginName, key := splitKey(key)
	if pluginName != PluginNameBase {
		// Set option on plugin
		plugin := app.plugins.GetByName(pluginName)
		if plugin == nil {
			return "Invalid plugin name: " + pluginName
		}
		log.Info("SetOption on plugin", "plugin", pluginName, "key", key, "value", value)
		return plugin.SetOption(app.state, key, value)
	} else {
		// Set option on hubcoin
		switch key {
		case "chainID":
			app.state.SetChainID(value)
			return "Success"
		case "account":
			var err error
			var acc *types.Account
			wire.ReadJSONPtr(&acc, []byte(value), &err)
			if err != nil {
				return "Error decoding acc message: " + err.Error()
			}
			app.state.SetAccount(acc.PubKey.Address(), acc)
			log.Info("SetAccount", "addr", acc.PubKey.Address(), "acc", acc)
			return "Success"
		}
		return "Unrecognized option key " + key
	}
}

// ABCI::DeliverTx
func (app *Hubcoin) DeliverTx(txBytes []byte) (res abci.Result) {
	if len(txBytes) > maxTxSize {
		return abci.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}

	// Decode tx
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	// Validate and exec tx
	res = sm.ExecTx(app.state, app.plugins, tx, false, nil)
	if res.IsErr() {
		return res.PrependLog("Error in DeliverTx")
	}
	return res
}

// ABCI::CheckTx
func (app *Hubcoin) CheckTx(txBytes []byte) (res abci.Result) {
	if len(txBytes) > maxTxSize {
		return abci.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}

	// Decode tx
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	// Validate tx
	res = sm.ExecTx(app.cacheState, app.plugins, tx, true, nil)
	if res.IsErr() {
		return res.PrependLog("Error in CheckTx")
	}
	return abci.OK
}

// ABCI::Query
func (app *Hubcoin) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	if len(reqQuery.Data) == 0 {
		resQuery.Log = "Query cannot be zero length"
		resQuery.Code = abci.CodeType_EncodingError
		return
	}

	// handle special path for account info
	if reqQuery.Path == "/account" {
		reqQuery.Path = "/key"
		reqQuery.Data = append([]byte("hub/a/"), reqQuery.Data...)
	}

	resQuery, err := app.eyesCli.QuerySync(reqQuery)
	if err != nil {
		resQuery.Log = "Failed to query MerkleEyes: " + err.Error()
		resQuery.Code = abci.CodeType_InternalError
		return
	}
	return
}

// ABCI::Commit
func (app *Hubcoin) Commit() (res abci.Result) {

	// Commit state
	res = app.state.Commit()

	// Wrap the committed state in cache for CheckTx
	app.cacheState = app.state.CacheWrap()

	if res.IsErr() {
		PanicSanity("Error getting hash: " + res.Error())
	}
	return res
}

// ABCI::InitChain
func (app *Hubcoin) InitChain(validators []*abci.Validator) {
	for _, plugin := range app.plugins.GetList() {
		plugin.InitChain(app.state, validators)
	}
}

// ABCI::BeginBlock
func (app *Hubcoin) BeginBlock(hash []byte, header *abci.Header) {
	for _, plugin := range app.plugins.GetList() {
		plugin.BeginBlock(app.state, hash, header)
	}
}

// ABCI::EndBlock
func (app *Hubcoin) EndBlock(height uint64) (res abci.ResponseEndBlock) {
	for _, plugin := range app.plugins.GetList() {
		pluginRes := plugin.EndBlock(app.state, height)
		res.Diffs = append(res.Diffs, pluginRes.Diffs...)
	}
	return
}

//----------------------------------------

// Splits the string at the first '/'.
// if there are none, the second string is nil.
func splitKey(key string) (prefix string, suffix string) {
	if strings.Contains(key, "/") {
		keyParts := strings.SplitN(key, "/", 2)
		return keyParts[0], keyParts[1]
	}
	return key, ""
}
