package main

import (
  "testing"
  "encoding/json"
)

func TestConfig(t *testing.T) {

    if isJSON(JsonCoinConfig) == false {
      t.Fail()
    }
}


func isJSON(s string) bool {
    var js map[string]interface{}
    return json.Unmarshal([]byte(s), &js) == nil

}
