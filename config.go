package main

// JsonCoinConfig contains coin specific json
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
         "blocktime":300
        },
        {
         "name": "BitCloud",
         "tag":"btdx",
         "binary":"bitcloud-cli",
         "cmdchain": [
           "./%b getblockcount",
           "./%b getblockhash %0",
           "./%b getblockheader %0" ],
         "blocktime":300
       },
       {
        "name": "EliteCoin",
        "tag":"1337",
        "binary":"1337d",
        "cmdchain": [
          "./%b getblockcount",
          "./%b getblockhash %0",
          "./%b getblock %0" ],
        "blocktime":300
      }
    ]
 }
`
