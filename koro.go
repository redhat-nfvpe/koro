/*
koro: Container ROuting tool
*/
package main

import (
	"fmt"
	"os"
	"net"
	"github.com/MakeNowJust/heredoc"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"strconv"
	"strings"
	"github.com/redhat-nfvpe/koro/parser"
	koko_api "github.com/redhat-nfvpe/koko/api"
)

// getNamepace identify namespace from given cli option
func getNamepace (command *parser.Command) (namespace string, err error) {
	switch command.TargetType {
	case parser.DOCKER:
		namespace, err = koko_api.GetDockerContainerNS(command.Target)
	case parser.IPNETNS:
		namespace = fmt.Sprintf("/var/run/netns/%s", command.Target)
	case parser.NETNS:
		namespace = command.Target
	case parser.PID:
		var pid int
		pid, err = strconv.Atoi(command.Target)
		if err == nil {
	            namespace = fmt.Sprintf("/proc/%d/ns/net", pid)
	        }
	}
	return namespace, err
}

// GetNetlinkRoute converts from CLI argument to netlink.Route structure
func GetNetlinkRoute (command *parser.Command) (route netlink.Route, err error) {
	var optionDevIfIndex int
	var optionViaAddress net.IP

	optionViaAddress = net.ParseIP(command.OptionVia)
	if optionViaAddress == nil {
		fmt.Fprintf(os.Stderr, "Failed to parse IP")
	}

	if command.OptionDev == "" {
		routeToViaIP, err1 := netlink.RouteGet(optionViaAddress)
		if err1 != nil {
			fmt.Fprintf(os.Stderr, "%v", err1)
		}
		if len(routeToViaIP) == 0 {
			fmt.Fprintf(os.Stderr, "no Route to via address")
		}
		optionDevIfIndex = routeToViaIP[0].LinkIndex
	} else {
		optionDevIf, err2 := netlink.LinkByName(command.OptionDev)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "%v", err2)
		}
		optionDevIfIndex = optionDevIf.Attrs().Index
	}

	if command.IsDefault {
		route = netlink.Route{
			LinkIndex: optionDevIfIndex,
			Gw: optionViaAddress,
			Dst: nil,
		}
	} else {
		network, netmask, err3 := net.ParseCIDR(
			fmt.Sprintf("%s/%s", command.Network, command.NetworkLength))
		if err3 != nil {
			return route, err3
		}

		ipnet := net.IPNet {
			IP: network,
			Mask: netmask.Mask,
		}
		route = netlink.Route{
			LinkIndex: optionDevIfIndex,
			Dst: &ipnet,
			Gw: optionViaAddress,
		}
	}

	return route, nil
}

// AddDelRoute does actuall operation to add/del route with netlink API
func AddDelRoute (command *parser.Command) (err error) {
	var (
		nsName string
		targetNS ns.NetNS
	)

	if command.TargetType == parser.NSNONE {
		targetNS, err = ns.GetCurrentNS()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return err
		}
	} else {
		nsName, err = getNamepace(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return err
		}
		targetNS, err = ns.GetNS(nsName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return err
		}
		defer targetNS.Close()
	}

	err = targetNS.Do(func(_ ns.NetNS) error {
		route, err1 := GetNetlinkRoute(command)
		if err1 != nil {
			return err1
		}
		switch command.Operation {
		case parser.ROUTEADD :
			if err2 := netlink.RouteAdd(&route); err2 != nil {
				return err2
			}
		case parser.ROUTEDEL:
			if err2 := netlink.RouteDel(&route); err2 != nil {
				return err2
			}
		}
		// call netlink.RouteAdd
		// add 1.1.1.0/24 via 192.168.1.1
		// add 1.1.2.0/24 dev eth0
		// add 1.1.3.0/24 via 192.168.1.1 dev eth0
		return nil
	})
	return nil
}

// AddDelAddr adds/deletes address with netlink API
func AddDelAddr (command *parser.Command) (err error) {
	var (
		nsName string
		targetNS ns.NetNS
	)

	if command.TargetType == parser.NSNONE {
		targetNS, err = ns.GetCurrentNS()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return err
		}
	} else {
		nsName, err = getNamepace(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return err
		}
		targetNS, err = ns.GetNS(nsName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return err
		}
		defer targetNS.Close()
	}

	if command.OptionVia != "" {
		return fmt.Errorf("address command does not support via keyword")
	}
	//Add check (no via, dev is valid)
	ip, mask, err := net.ParseCIDR(
		fmt.Sprintf("%s/%s", command.Network, command.NetworkLength))
	if err != nil {
		return err
	}

	err = targetNS.Do(func(_ ns.NetNS) error {
		optionDevIf, err2 := netlink.LinkByName(command.OptionDev)
		if err2 != nil {
			return err2
		}
		addr := &netlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: mask.Mask}, Label: ""}

		switch command.Operation {
		case parser.ADDRADD:
			if err3 := netlink.AddrAdd(optionDevIf, addr); err3 != nil {
				return fmt.Errorf("failed to add IP addr %v to %q: %v",
                                        addr, command.OptionDev, err3)
			}
		case parser.ADDRDEL:
			if err3 := netlink.AddrDel(optionDevIf, addr); err3 != nil {
				return fmt.Errorf("failed to delete IP addr %v to %q: %v",
                                        addr, command.OptionDev, err3)
			}
		}
		return nil
	})
	return nil
}

// usage shows usage when user does not provide any arguments
func usage() {
	doc := heredoc.Doc(`
		Example:
		./koro docker <name> address add 127.0.0.3/24 dev lo
	`)
	fmt.Print(doc)
}

func main () {
	args := strings.Join(os.Args[1:], " ")
	if args == "" {
		usage()
		os.Exit(0)
	}
	p := parser.ParseCommand(args)

	c := p.GetCommand()
	switch c.Operation {
	case parser.ROUTEADD, parser.ROUTEDEL:
		if err := AddDelRoute(c); err != nil {
			fmt.Fprintf(os.Stderr, "err:%v", err)
		} else {
			fmt.Println("Succeed!")
		}
	case parser.ADDRADD, parser.ADDRDEL:
		if err := AddDelAddr(c); err != nil {
			fmt.Fprintf(os.Stderr, "err:%v", err)
		} else {
			fmt.Println("Succeed!")
		}
	}
}
