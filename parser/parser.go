package parser

import (
    "fmt"
    // "strconv"
    "strings"
)

const (
	ROUTEADD = iota
	ROUTEDEL
	ADDRADD
	ADDRDEL
	VIA
	DEV
)

const (
	DOCKER = iota
	NETNS
	PID
	NSNONE
)

type Command struct {
    Operation   int
    TargetType  int
    Target      string
    IsError     bool
    IsDefault	bool
    Network	string
    NetworkLength	string
    OptionVia   string
    OptionDev   string
}

func (c *Command) GetCommand() (*Command) {
	return c
}

func (c *Command) Dump() {
    fmt.Printf("IsError:%v\n", c.IsError)
    fmt.Printf("TargetType:%d\n", c.TargetType)
    fmt.Printf("Target:%s\n", c.Target)
    fmt.Printf("Operation:%d\n", c.Operation)
    fmt.Printf("Network:%s\n", c.Network)
    fmt.Printf("NetworkLength:%s\n", c.NetworkLength)
    fmt.Printf("Via:%s\n", c.OptionVia)
    fmt.Printf("Dev:%s\n", c.OptionDev)
}

func (c *Command) SetOption(name string, val string) {
	switch name {
	case "via":
		c.OptionVia = val
	case "dev":
		c.OptionDev = val
	}
}

func (c *Command) Err(pos int, buffer string) {
    fmt.Println("")
    a := strings.Split(buffer[:pos], "\n")
    row := len(a) - 1
    column := len(a[row]) - 1

    lines := strings.Split(buffer, "\n")
    for i := row - 5; i <= row; i++ {
        if i < 0 {
            i = 0
        }

        fmt.Println(lines[i])
    }

    s := ""
    for i := 0; i <= column; i++ {
        s += " "
    }
    ln := len(strings.Trim(lines[row], " \r\n"))
    for i := column + 1; i < ln; i++ {
        s += "~"
    }
    fmt.Println(s)

    fmt.Println("error")
    c.IsError = true
}

func ParseCommand (command string) (p Parser) {
    p = Parser{Buffer: command}
    p.Init()
    err := p.Parse()
    if err != nil {
        fmt.Println(err)
    } else {
	p.Execute()
	//debug code
	/*
        if p.IsError == false {
            //fmt.Println("parsed!")
        }
	*/
    }
    return p
}
