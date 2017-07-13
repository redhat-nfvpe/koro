package main

import (
	"testing"
)
import	"./parser"

func TestGetNetlinkRoute(t *testing.T) {
	command1 := parser.Command{
		Operation: parser.ROUTEADD,
		Network: "192.168.1.0",
		NetworkLength: "24",
		OptionVia: "127.0.0.1",
		OptionDev: "lo",
	}
	route, err1 := GetNetlinkRoute(&command1)
	if (err1 != nil || route.LinkIndex == 0) {
		t.Fatalf("Parse error: %v/%v", route, err1)
	}

}
