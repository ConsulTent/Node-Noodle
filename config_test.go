package main

import (
  "testing"
  "encoding/json"
)

func TestConfig(t *testing.T) {

  if func (s string) bool {
      var js map[string]interface{}

      return json.Unmarshal([]byte(s), &js) == nil
  }(JsonCoinConfig) == false {
    t.Fail()
  }
}
