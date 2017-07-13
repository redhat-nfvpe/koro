package parser

import "testing"

func TestParseCommand (t *testing.T) {
	test1 := "docker testDocker route add 10.1.1.0/24 via 10.1.1.1"
	if p := ParseCommand(test1);
	   p.TargetType != DOCKER ||
	   p.Target != "testDocker" ||
	   p.Operation != ROUTEADD ||
	   p.Network != "10.1.1.0" ||
	   p.NetworkLength != "24" ||
	   p.OptionVia != "10.1.1.1" {
		   p.Dump()
		   t.Fatalf("failed at parsing: %s", test1)
	   }
	test2 := "docker testDocker route del 10.1.1.0/24"
	if p := ParseCommand(test2);
	   p.TargetType != DOCKER ||
	   p.Target != "testDocker" ||
	   p.Operation != ROUTEDEL ||
	   p.Network != "10.1.1.0" ||
	   p.NetworkLength != "24" {
		   p.Dump()
		   t.Fatalf("failed at parsing: %s", test2)
	   }
}
