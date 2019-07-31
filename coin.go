package main

import (
	"strconv"
	"fmt"
	"os"
	"strings"
	"time"
	"net/http"
	"io/ioutil"
//	"bytes"

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
	Time        int64
	Blocks      int64
	InsightBlocks	int64
	CaptureTime int64
	Detected    bool
	InsightUrl	string
	InsightStatus	string
	InsightFormat	string
	InsightKey	string
}



// GetBlockInsightHeight Grab blockheight from outside url
func GetBlockInsightHeight() string {
	var Query string
	var result string

Query = strings.ReplaceAll(Coin.InsightUrl, "%0", Coin.InsightStatus)

/*
		 n := bytes.IndexByte(body, 0)
		 log.Debug(fmt.Sprintf("Insight body size: %v", n))
		 result = gjson.Get(string(body[:n]), "blockheight").String()
*/

		   result = NetData(Query)
//		 result = gjson.Get(NetData(Query), "info.blocks").String()

		if Coin.InsightFormat == "json" {
					 log.Debug(fmt.Sprintf("Insight json result: %s",result))
		 result = gjson.Get(result, Coin.InsightKey).String()
	  } else {
			log.Debug(fmt.Sprintf("Insight text result: %s",result))
		}

		blocks , err := strconv.Atoi(result)
		 if err != nil {
			 log.Debug(fmt.Sprintf("GetBlockInsightHeight conversion error: %v",err))
			 return "0"
		 }
		 Coin.InsightBlocks = int64(blocks)

	 return result
}

// ShowCoins() lists all available coins in the config.
func ShowCoins() {

	coins := gjson.Get(JsonCoinConfig, "coins")
	log.Debug(fmt.Sprintf("ShowCoins(): coins: %s", coins))
	for i, data := range coins.Array() {
		log.Debug(fmt.Sprintf("coins.Array():i: %d, data: %s", i, data))
		log.Info(fmt.Sprintf("Supported: %s\n", gjson.Get(fmt.Sprintf("%s", data), "name").String()))

	}
}

// DetectCoin() Runs through the binary list and checks for existance
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
			Coin.BlockTime = int64(gjson.Get(fmt.Sprintf("%s", data), "blocktime").Int())
			coinarray := gjson.Get(fmt.Sprintf("%s", data), "cmdchain").Array()
	// Let's add Insight config
			Coin.InsightUrl = gjson.Get(fmt.Sprintf("%s", data), "insight.baseurl").String()
			Coin.InsightStatus = gjson.Get(fmt.Sprintf("%s", data), "insight.status").String()
			Coin.InsightFormat = gjson.Get(fmt.Sprintf("%s", data), "insight.format").String()
			if Coin.InsightFormat == "json" {
				Coin.InsightKey = gjson.Get(fmt.Sprintf("%s", data), "insight.key").String()
			} else {
				Coin.InsightKey = ""
			}
	// ***
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
	Coin.Time = blocktime.Int()
	Coin.CaptureTime = time.Now().Unix()

}

func BlocksBehind() int64 {

 blocks , err := strconv.Atoi(GetBlockInsightHeight())
  if err != nil {
		log.Debug(fmt.Sprintf("BlocksBehind conversion error: %v",err))
		return int64(0)
	}

	return int64(blocks) - Coin.Blocks
}

func PrintBlocksBehind(v bool) {


		bb := BlocksBehind()
		if bb > 15 {
			log.Warn(fmt.Sprintf("We are %d blocks behind network!",bb))
		} else {
			if v == true {
				log.Info(fmt.Sprintf("We are %d blocks behind network.",bb))
			}
		}
}


// NetData Pull insight data
func NetData(q string) string {
	var result string

	tr := &http.Transport{
	MaxIdleConns:       10,
	IdleConnTimeout:    30 * time.Second,
	DisableCompression: false,
 }

var NetClient = &http.Client{
		Transport: tr,
	}

	log.Debug(fmt.Sprintf("NetData Query url: %s",q))

	req, err := http.NewRequest("GET", q, nil)
			 if err != nil {
							 log.Debug(fmt.Sprintf("NetData NewRequest failed: %v",err))
							 return ""
			 }

	req.Header.Set("User-Agent", fmt.Sprintf("Node-Noodle/%s",pver))


	resp, err := NetClient.Do(req)
	if err != nil {
		log.Debug(fmt.Sprintf("Insight Query error: %v",err))
		result = "0"
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debug(fmt.Sprintf("Insight Read error: %v",err))
		result = "0"
	} else {
			 log.Debug(fmt.Sprintf("Insight body: %s", string(body)))
			 result = string(body)
  }
	return result
}
