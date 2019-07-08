package main

import (
	//	"strconv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	// "sync"
)

// GenericCoin Coin structure
type GenericCoin struct {
	Name        string   `json:"name"`
	Tag         string   `json:"tag"`
	Binary      string   `json:"binary"`
	CmdChain    []string `json:"cmdchain"`
	BlockTime   int64    `json:"blocktime"`
	Blocks      int64
	CaptureTime int64
	Detected    bool
}

func ShowCoins() {

	coins := gjson.Get(JsonCoinConfig, "coins")
	log.Debug(fmt.Sprintf("ShowCoins(): coins: %s", coins))
	for i, data := range coins.Array() {
		log.Debug(fmt.Sprintf("coins.Array():i: %d, data: %s", i, data))
		log.Info(fmt.Sprintf("Supported: %s\n", gjson.Get(fmt.Sprintf("%s", data), "name").String()))

	}
}

func DetectCoin() bool {
	var cn gjson.Result
	var detected bool
	var err error

	coins := gjson.Get(JsonCoinConfig, "coins")

	detected = false

	for id, data := range coins.Array() {
		log.Debug(fmt.Sprintf("id: %d, data: %s", id, data))
		cn = gjson.Get(fmt.Sprintf("%s", data), "binary")
		_, err = os.Stat(cn.String())
		if os.IsNotExist(err) {
			log.Debug("binary Not Detected")
		} else {
			log.Debug("binary Detected")
			Coin.Binary = cn.String()
			Coin.Name = gjson.Get(fmt.Sprintf("%s", data), "name").String()
			Coin.Tag = gjson.Get(fmt.Sprintf("%s", data), "tag").String()
			coinarray := gjson.Get(fmt.Sprintf("%s", data), "cmdchain").Array()
			log.Debug(fmt.Sprintf("\ncoinarray: %+v\n", coinarray))
			detected = true
			for _, ca := range coinarray {
				Coin.CmdChain = append(Coin.CmdChain, ca.String())
			}
			//		Coin.Cmdchain[0] = coinarray[0].String()
			Coin.Detected = true
		}
		log.Debug(fmt.Sprintf("Coin: %+v\n", Coin))
		//		fmt.Printf("Cmdchain[0]: %s\n", Coin.Cmdchain[2])
		//		println(fmt.Sprintf("d: %s", d.Str))
	}
	return detected
}

func InitCoin() {
	var output string
	var command string

	log.Debug("In initCoin.")

	// Setup the commands

	for i, data := range Coin.CmdChain {

		if i > 0 {
			command = strings.ReplaceAll(data, "%b", Coin.Binary)
			command = strings.ReplaceAll(command, "%0", output)
		} else {
			command = strings.ReplaceAll(data, "%b", Coin.Binary)
		}

		output = exe_cmd(command)
	}

	log.Debug(fmt.Sprintf("block header %s", output))

	//    log.Debug(fmt.Sprintf("%s",blockchaininfo))

	blocks := gjson.Get(output, "height")
	blocktime := gjson.Get(output, "time")

	log.Debug(fmt.Sprintf("json: blocks: %s", blocks))

	log.Debug(fmt.Sprintf("json: block time %s", blocktime))

	Coin.Blocks = blocks.Int()
	Coin.BlockTime = blocktime.Int()
	Coin.CaptureTime = time.Now().Unix()

}
