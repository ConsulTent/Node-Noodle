package main

/*
   JsonCoinConfig contains coin specific json
   This is where you'd want to add new coins.
*/

var JsonCoinConfig string = `
{ "coins": [
        {
         "name": "ZCoin",
         "tag":"xzc",
         "binary":"zcoin-cli",
         "cmdchain": [
           "./%b getblockcount",
           "./%b getblockhash %0",
           "./%b getblockheader %0" ],
         "blocktime":300,
         "insight": {
           "baseurl":"https://explorer.zcoin.io/api/%0",
           "status":"/status"
         }
        },
        {
         "name": "BitCloud",
         "tag":"btdx",
         "binary":"bitcloud-cli",
         "cmdchain": [
           "./%b getblockcount",
           "./%b getblockhash %0",
           "./%b getblockheader %0" ],
         "blocktime":300,
         "insight": {
           "baseurl":"https://chainz.cryptoid.info/btdx/api.dws?q=%0",
           "status":"getblockcount"
         }
       },
       {
        "name": "EliteCoin",
        "tag":"1337",
        "binary":"1337d",
        "cmdchain": [
          "./%b getblockcount",
          "./%b getblockhash %0",
          "./%b getblock %0" ],
        "blocktime":60,
        "insight": {
          "baseurl":"https://chainz.cryptoid.info/1337/api.dws?q=%0",
          "status":"getblockcount"
        }
      },
      {
       "name": "Dash",
       "tag":"DASH",
       "binary":"dash-cli",
       "cmdchain": [
         "./%b getblockcount",
         "./%b getblockhash %0",
         "./%b getblockheader %0" ],
       "blocktime":160,
       "insight": {
         "baseurl":"https://insight.dash.org/api/%0",
         "status":"/status"
       }
     },
     {
       "name": "PIVX",
       "tag":"PIVX",
       "binary":"pivx-cli",
       "cmdchain": [
         "./%b getblockcount",
         "./%b getblockhash %0",
         "./%b getblockheader %0" ],
       "blocktime":60,
       "insight": {
         "baseurl":"https://chainz.cryptoid.info/pivx/api.dws?q=%0",
         "status":"getblockcount"
       }
     }
    ]
 }
`
