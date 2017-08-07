#! /bin/bash

killall -9 hubcoin tendermint
TMROOT=./data/chain1/tendermint tendermint unsafe_reset_all
TMROOT=./data/chain2/tendermint tendermint unsafe_reset_all

rm -rf ./data/chain1/hubcoin/merkleeyes.db
rm -rf ./data/chain2/hubcoin/merkleeyes.db

rm ./*.log

rm ./data/chain1/tendermint/*.bak
rm ./data/chain2/tendermint/*.bak
