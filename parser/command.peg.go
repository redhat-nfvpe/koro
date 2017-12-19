package parser

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleroot
	ruleEOT
	rulenetns
	rulenetnsid
	ruleoperation
	rulenetwork
	ruleaddrstr
	rulelen
	ruleoption
	rulespaces
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
	ruleAction22
	ruleAction23
	ruleAction24
	ruleAction25
	ruleAction26
)

var rul3s = [...]string{
	"Unknown",
	"root",
	"EOT",
	"netns",
	"netnsid",
	"operation",
	"network",
	"addrstr",
	"len",
	"option",
	"spaces",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
	"Action22",
	"Action23",
	"Action24",
	"Action25",
	"Action26",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type Parser struct {
	Command

	Buffer string
	buffer []rune
	rules  [39]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *Parser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *Parser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *Parser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *Parser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *Parser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.Err(begin, buffer, "")
		case ruleAction1:
			p.TargetType = NSNONE
		case ruleAction2:
			p.Err(begin, buffer, "")
		case ruleAction3:
			p.TargetType = DOCKER
		case ruleAction4:
			p.TargetType = NETNS
		case ruleAction5:
			p.TargetType = PID
		case ruleAction6:
			p.Err(begin, buffer, "Invalid namespace")
		case ruleAction7:
			p.Target = text
		case ruleAction8:
			p.Operation = ROUTEADD
		case ruleAction9:
			p.Operation = ROUTEDEL
		case ruleAction10:
			p.Err(begin, buffer, "Invalid option")
		case ruleAction11:
			p.Err(begin, buffer, "invalid network")
		case ruleAction12:
			p.Err(begin, buffer, "Invalid option")
		case ruleAction13:
			p.Err(begin, buffer, "Invalid network")
		case ruleAction14:
			p.Err(begin, buffer, "")
		case ruleAction15:
			p.Operation = ADDRADD
		case ruleAction16:
			p.Operation = ADDRDEL
		case ruleAction17:
			p.Err(begin, buffer, "Invalid option")
		case ruleAction18:
			p.Err(begin, buffer, "Invalid address")
		case ruleAction19:
			p.Err(begin, buffer, "Invalid option")
		case ruleAction20:
			p.Err(begin, buffer, "Invalid address")
		case ruleAction21:
			p.IsDefault = false
		case ruleAction22:
			p.IsDefault = true
		case ruleAction23:
			p.Network = text
		case ruleAction24:
			p.NetworkLength = text
		case ruleAction25:
			p.SetOption("via", text)
		case ruleAction26:
			p.SetOption("dev", text)

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *Parser) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 root <- <((netns spaces operation EOT) / (netns spaces <.+> Action0 EOT) / (operation Action1 EOT) / (<.+> Action2 EOT))> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				{
					position2, tokenIndex2 := position, tokenIndex
					if !_rules[rulenetns]() {
						goto l3
					}
					if !_rules[rulespaces]() {
						goto l3
					}
					if !_rules[ruleoperation]() {
						goto l3
					}
					if !_rules[ruleEOT]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex = position2, tokenIndex2
					if !_rules[rulenetns]() {
						goto l4
					}
					if !_rules[rulespaces]() {
						goto l4
					}
					{
						position5 := position
						if !matchDot() {
							goto l4
						}
					l6:
						{
							position7, tokenIndex7 := position, tokenIndex
							if !matchDot() {
								goto l7
							}
							goto l6
						l7:
							position, tokenIndex = position7, tokenIndex7
						}
						add(rulePegText, position5)
					}
					if !_rules[ruleAction0]() {
						goto l4
					}
					if !_rules[ruleEOT]() {
						goto l4
					}
					goto l2
				l4:
					position, tokenIndex = position2, tokenIndex2
					if !_rules[ruleoperation]() {
						goto l8
					}
					if !_rules[ruleAction1]() {
						goto l8
					}
					if !_rules[ruleEOT]() {
						goto l8
					}
					goto l2
				l8:
					position, tokenIndex = position2, tokenIndex2
					{
						position9 := position
						if !matchDot() {
							goto l0
						}
					l10:
						{
							position11, tokenIndex11 := position, tokenIndex
							if !matchDot() {
								goto l11
							}
							goto l10
						l11:
							position, tokenIndex = position11, tokenIndex11
						}
						add(rulePegText, position9)
					}
					if !_rules[ruleAction2]() {
						goto l0
					}
					if !_rules[ruleEOT]() {
						goto l0
					}
				}
			l2:
				add(ruleroot, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 EOT <- <!.> */
		func() bool {
			position12, tokenIndex12 := position, tokenIndex
			{
				position13 := position
				{
					position14, tokenIndex14 := position, tokenIndex
					if !matchDot() {
						goto l14
					}
					goto l12
				l14:
					position, tokenIndex = position14, tokenIndex14
				}
				add(ruleEOT, position13)
			}
			return true
		l12:
			position, tokenIndex = position12, tokenIndex12
			return false
		},
		/* 2 netns <- <(('d' 'o' 'c' 'k' 'e' 'r' spaces netnsid Action3) / ('n' 'e' 't' 'n' 's' spaces netnsid Action4) / ('p' 'i' 'd' spaces netnsid Action5) / (<.+> Action6 EOT))> */
		func() bool {
			position15, tokenIndex15 := position, tokenIndex
			{
				position16 := position
				{
					position17, tokenIndex17 := position, tokenIndex
					if buffer[position] != rune('d') {
						goto l18
					}
					position++
					if buffer[position] != rune('o') {
						goto l18
					}
					position++
					if buffer[position] != rune('c') {
						goto l18
					}
					position++
					if buffer[position] != rune('k') {
						goto l18
					}
					position++
					if buffer[position] != rune('e') {
						goto l18
					}
					position++
					if buffer[position] != rune('r') {
						goto l18
					}
					position++
					if !_rules[rulespaces]() {
						goto l18
					}
					if !_rules[rulenetnsid]() {
						goto l18
					}
					if !_rules[ruleAction3]() {
						goto l18
					}
					goto l17
				l18:
					position, tokenIndex = position17, tokenIndex17
					if buffer[position] != rune('n') {
						goto l19
					}
					position++
					if buffer[position] != rune('e') {
						goto l19
					}
					position++
					if buffer[position] != rune('t') {
						goto l19
					}
					position++
					if buffer[position] != rune('n') {
						goto l19
					}
					position++
					if buffer[position] != rune('s') {
						goto l19
					}
					position++
					if !_rules[rulespaces]() {
						goto l19
					}
					if !_rules[rulenetnsid]() {
						goto l19
					}
					if !_rules[ruleAction4]() {
						goto l19
					}
					goto l17
				l19:
					position, tokenIndex = position17, tokenIndex17
					if buffer[position] != rune('p') {
						goto l20
					}
					position++
					if buffer[position] != rune('i') {
						goto l20
					}
					position++
					if buffer[position] != rune('d') {
						goto l20
					}
					position++
					if !_rules[rulespaces]() {
						goto l20
					}
					if !_rules[rulenetnsid]() {
						goto l20
					}
					if !_rules[ruleAction5]() {
						goto l20
					}
					goto l17
				l20:
					position, tokenIndex = position17, tokenIndex17
					{
						position21 := position
						if !matchDot() {
							goto l15
						}
					l22:
						{
							position23, tokenIndex23 := position, tokenIndex
							if !matchDot() {
								goto l23
							}
							goto l22
						l23:
							position, tokenIndex = position23, tokenIndex23
						}
						add(rulePegText, position21)
					}
					if !_rules[ruleAction6]() {
						goto l15
					}
					if !_rules[ruleEOT]() {
						goto l15
					}
				}
			l17:
				add(rulenetns, position16)
			}
			return true
		l15:
			position, tokenIndex = position15, tokenIndex15
			return false
		},
		/* 3 netnsid <- <(<(!' ' .)+> Action7)> */
		func() bool {
			position24, tokenIndex24 := position, tokenIndex
			{
				position25 := position
				{
					position26 := position
					{
						position29, tokenIndex29 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l29
						}
						position++
						goto l24
					l29:
						position, tokenIndex = position29, tokenIndex29
					}
					if !matchDot() {
						goto l24
					}
				l27:
					{
						position28, tokenIndex28 := position, tokenIndex
						{
							position30, tokenIndex30 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l30
							}
							position++
							goto l28
						l30:
							position, tokenIndex = position30, tokenIndex30
						}
						if !matchDot() {
							goto l28
						}
						goto l27
					l28:
						position, tokenIndex = position28, tokenIndex28
					}
					add(rulePegText, position26)
				}
				if !_rules[ruleAction7]() {
					goto l24
				}
				add(rulenetnsid, position25)
			}
			return true
		l24:
			position, tokenIndex = position24, tokenIndex24
			return false
		},
		/* 4 operation <- <(('r' 'o' 'u' 't' 'e' spaces ('a' 'd' 'd') spaces network (spaces option)* Action8) / ('r' 'o' 'u' 't' 'e' spaces ('d' 'e' 'l') spaces network (spaces option)* Action9) / ('r' 'o' 'u' 't' 'e' spaces ('a' 'd' 'd') spaces network spaces <.+> Action10 EOT) / ('r' 'o' 'u' 't' 'e' spaces ('a' 'd' 'd') spaces <.+> Action11 EOT) / ('r' 'o' 'u' 't' 'e' spaces ('d' 'e' 'l') spaces network spaces <.+> Action12 EOT) / ('r' 'o' 'u' 't' 'e' spaces ('d' 'e' 'l') spaces <.+> Action13 EOT) / ('r' 'o' 'u' 't' 'e' spaces <.+> Action14 EOT) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('a' 'd' 'd') spaces network spaces option Action15) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('d' 'e' 'l') spaces network spaces option Action16) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('a' 'd' 'd') spaces network spaces <.+> Action17 EOT) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('a' 'd' 'd') spaces <.+> Action18 EOT) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('d' 'e' 'l') spaces network spaces <.+> Action19 EOT) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('d' 'e' 'l') spaces <.+> Action20 EOT) / )> */
		func() bool {
			{
				position32 := position
				{
					position33, tokenIndex33 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l34
					}
					position++
					if buffer[position] != rune('o') {
						goto l34
					}
					position++
					if buffer[position] != rune('u') {
						goto l34
					}
					position++
					if buffer[position] != rune('t') {
						goto l34
					}
					position++
					if buffer[position] != rune('e') {
						goto l34
					}
					position++
					if !_rules[rulespaces]() {
						goto l34
					}
					if buffer[position] != rune('a') {
						goto l34
					}
					position++
					if buffer[position] != rune('d') {
						goto l34
					}
					position++
					if buffer[position] != rune('d') {
						goto l34
					}
					position++
					if !_rules[rulespaces]() {
						goto l34
					}
					if !_rules[rulenetwork]() {
						goto l34
					}
				l35:
					{
						position36, tokenIndex36 := position, tokenIndex
						if !_rules[rulespaces]() {
							goto l36
						}
						if !_rules[ruleoption]() {
							goto l36
						}
						goto l35
					l36:
						position, tokenIndex = position36, tokenIndex36
					}
					if !_rules[ruleAction8]() {
						goto l34
					}
					goto l33
				l34:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('r') {
						goto l37
					}
					position++
					if buffer[position] != rune('o') {
						goto l37
					}
					position++
					if buffer[position] != rune('u') {
						goto l37
					}
					position++
					if buffer[position] != rune('t') {
						goto l37
					}
					position++
					if buffer[position] != rune('e') {
						goto l37
					}
					position++
					if !_rules[rulespaces]() {
						goto l37
					}
					if buffer[position] != rune('d') {
						goto l37
					}
					position++
					if buffer[position] != rune('e') {
						goto l37
					}
					position++
					if buffer[position] != rune('l') {
						goto l37
					}
					position++
					if !_rules[rulespaces]() {
						goto l37
					}
					if !_rules[rulenetwork]() {
						goto l37
					}
				l38:
					{
						position39, tokenIndex39 := position, tokenIndex
						if !_rules[rulespaces]() {
							goto l39
						}
						if !_rules[ruleoption]() {
							goto l39
						}
						goto l38
					l39:
						position, tokenIndex = position39, tokenIndex39
					}
					if !_rules[ruleAction9]() {
						goto l37
					}
					goto l33
				l37:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('r') {
						goto l40
					}
					position++
					if buffer[position] != rune('o') {
						goto l40
					}
					position++
					if buffer[position] != rune('u') {
						goto l40
					}
					position++
					if buffer[position] != rune('t') {
						goto l40
					}
					position++
					if buffer[position] != rune('e') {
						goto l40
					}
					position++
					if !_rules[rulespaces]() {
						goto l40
					}
					if buffer[position] != rune('a') {
						goto l40
					}
					position++
					if buffer[position] != rune('d') {
						goto l40
					}
					position++
					if buffer[position] != rune('d') {
						goto l40
					}
					position++
					if !_rules[rulespaces]() {
						goto l40
					}
					if !_rules[rulenetwork]() {
						goto l40
					}
					if !_rules[rulespaces]() {
						goto l40
					}
					{
						position41 := position
						if !matchDot() {
							goto l40
						}
					l42:
						{
							position43, tokenIndex43 := position, tokenIndex
							if !matchDot() {
								goto l43
							}
							goto l42
						l43:
							position, tokenIndex = position43, tokenIndex43
						}
						add(rulePegText, position41)
					}
					if !_rules[ruleAction10]() {
						goto l40
					}
					if !_rules[ruleEOT]() {
						goto l40
					}
					goto l33
				l40:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('r') {
						goto l44
					}
					position++
					if buffer[position] != rune('o') {
						goto l44
					}
					position++
					if buffer[position] != rune('u') {
						goto l44
					}
					position++
					if buffer[position] != rune('t') {
						goto l44
					}
					position++
					if buffer[position] != rune('e') {
						goto l44
					}
					position++
					if !_rules[rulespaces]() {
						goto l44
					}
					if buffer[position] != rune('a') {
						goto l44
					}
					position++
					if buffer[position] != rune('d') {
						goto l44
					}
					position++
					if buffer[position] != rune('d') {
						goto l44
					}
					position++
					if !_rules[rulespaces]() {
						goto l44
					}
					{
						position45 := position
						if !matchDot() {
							goto l44
						}
					l46:
						{
							position47, tokenIndex47 := position, tokenIndex
							if !matchDot() {
								goto l47
							}
							goto l46
						l47:
							position, tokenIndex = position47, tokenIndex47
						}
						add(rulePegText, position45)
					}
					if !_rules[ruleAction11]() {
						goto l44
					}
					if !_rules[ruleEOT]() {
						goto l44
					}
					goto l33
				l44:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('r') {
						goto l48
					}
					position++
					if buffer[position] != rune('o') {
						goto l48
					}
					position++
					if buffer[position] != rune('u') {
						goto l48
					}
					position++
					if buffer[position] != rune('t') {
						goto l48
					}
					position++
					if buffer[position] != rune('e') {
						goto l48
					}
					position++
					if !_rules[rulespaces]() {
						goto l48
					}
					if buffer[position] != rune('d') {
						goto l48
					}
					position++
					if buffer[position] != rune('e') {
						goto l48
					}
					position++
					if buffer[position] != rune('l') {
						goto l48
					}
					position++
					if !_rules[rulespaces]() {
						goto l48
					}
					if !_rules[rulenetwork]() {
						goto l48
					}
					if !_rules[rulespaces]() {
						goto l48
					}
					{
						position49 := position
						if !matchDot() {
							goto l48
						}
					l50:
						{
							position51, tokenIndex51 := position, tokenIndex
							if !matchDot() {
								goto l51
							}
							goto l50
						l51:
							position, tokenIndex = position51, tokenIndex51
						}
						add(rulePegText, position49)
					}
					if !_rules[ruleAction12]() {
						goto l48
					}
					if !_rules[ruleEOT]() {
						goto l48
					}
					goto l33
				l48:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('r') {
						goto l52
					}
					position++
					if buffer[position] != rune('o') {
						goto l52
					}
					position++
					if buffer[position] != rune('u') {
						goto l52
					}
					position++
					if buffer[position] != rune('t') {
						goto l52
					}
					position++
					if buffer[position] != rune('e') {
						goto l52
					}
					position++
					if !_rules[rulespaces]() {
						goto l52
					}
					if buffer[position] != rune('d') {
						goto l52
					}
					position++
					if buffer[position] != rune('e') {
						goto l52
					}
					position++
					if buffer[position] != rune('l') {
						goto l52
					}
					position++
					if !_rules[rulespaces]() {
						goto l52
					}
					{
						position53 := position
						if !matchDot() {
							goto l52
						}
					l54:
						{
							position55, tokenIndex55 := position, tokenIndex
							if !matchDot() {
								goto l55
							}
							goto l54
						l55:
							position, tokenIndex = position55, tokenIndex55
						}
						add(rulePegText, position53)
					}
					if !_rules[ruleAction13]() {
						goto l52
					}
					if !_rules[ruleEOT]() {
						goto l52
					}
					goto l33
				l52:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('r') {
						goto l56
					}
					position++
					if buffer[position] != rune('o') {
						goto l56
					}
					position++
					if buffer[position] != rune('u') {
						goto l56
					}
					position++
					if buffer[position] != rune('t') {
						goto l56
					}
					position++
					if buffer[position] != rune('e') {
						goto l56
					}
					position++
					if !_rules[rulespaces]() {
						goto l56
					}
					{
						position57 := position
						if !matchDot() {
							goto l56
						}
					l58:
						{
							position59, tokenIndex59 := position, tokenIndex
							if !matchDot() {
								goto l59
							}
							goto l58
						l59:
							position, tokenIndex = position59, tokenIndex59
						}
						add(rulePegText, position57)
					}
					if !_rules[ruleAction14]() {
						goto l56
					}
					if !_rules[ruleEOT]() {
						goto l56
					}
					goto l33
				l56:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('a') {
						goto l60
					}
					position++
					if buffer[position] != rune('d') {
						goto l60
					}
					position++
					if buffer[position] != rune('d') {
						goto l60
					}
					position++
					if buffer[position] != rune('r') {
						goto l60
					}
					position++
					if buffer[position] != rune('e') {
						goto l60
					}
					position++
					if buffer[position] != rune('s') {
						goto l60
					}
					position++
					if buffer[position] != rune('s') {
						goto l60
					}
					position++
					if !_rules[rulespaces]() {
						goto l60
					}
					if buffer[position] != rune('a') {
						goto l60
					}
					position++
					if buffer[position] != rune('d') {
						goto l60
					}
					position++
					if buffer[position] != rune('d') {
						goto l60
					}
					position++
					if !_rules[rulespaces]() {
						goto l60
					}
					if !_rules[rulenetwork]() {
						goto l60
					}
					if !_rules[rulespaces]() {
						goto l60
					}
					if !_rules[ruleoption]() {
						goto l60
					}
					if !_rules[ruleAction15]() {
						goto l60
					}
					goto l33
				l60:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('a') {
						goto l61
					}
					position++
					if buffer[position] != rune('d') {
						goto l61
					}
					position++
					if buffer[position] != rune('d') {
						goto l61
					}
					position++
					if buffer[position] != rune('r') {
						goto l61
					}
					position++
					if buffer[position] != rune('e') {
						goto l61
					}
					position++
					if buffer[position] != rune('s') {
						goto l61
					}
					position++
					if buffer[position] != rune('s') {
						goto l61
					}
					position++
					if !_rules[rulespaces]() {
						goto l61
					}
					if buffer[position] != rune('d') {
						goto l61
					}
					position++
					if buffer[position] != rune('e') {
						goto l61
					}
					position++
					if buffer[position] != rune('l') {
						goto l61
					}
					position++
					if !_rules[rulespaces]() {
						goto l61
					}
					if !_rules[rulenetwork]() {
						goto l61
					}
					if !_rules[rulespaces]() {
						goto l61
					}
					if !_rules[ruleoption]() {
						goto l61
					}
					if !_rules[ruleAction16]() {
						goto l61
					}
					goto l33
				l61:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('a') {
						goto l62
					}
					position++
					if buffer[position] != rune('d') {
						goto l62
					}
					position++
					if buffer[position] != rune('d') {
						goto l62
					}
					position++
					if buffer[position] != rune('r') {
						goto l62
					}
					position++
					if buffer[position] != rune('e') {
						goto l62
					}
					position++
					if buffer[position] != rune('s') {
						goto l62
					}
					position++
					if buffer[position] != rune('s') {
						goto l62
					}
					position++
					if !_rules[rulespaces]() {
						goto l62
					}
					if buffer[position] != rune('a') {
						goto l62
					}
					position++
					if buffer[position] != rune('d') {
						goto l62
					}
					position++
					if buffer[position] != rune('d') {
						goto l62
					}
					position++
					if !_rules[rulespaces]() {
						goto l62
					}
					if !_rules[rulenetwork]() {
						goto l62
					}
					if !_rules[rulespaces]() {
						goto l62
					}
					{
						position63 := position
						if !matchDot() {
							goto l62
						}
					l64:
						{
							position65, tokenIndex65 := position, tokenIndex
							if !matchDot() {
								goto l65
							}
							goto l64
						l65:
							position, tokenIndex = position65, tokenIndex65
						}
						add(rulePegText, position63)
					}
					if !_rules[ruleAction17]() {
						goto l62
					}
					if !_rules[ruleEOT]() {
						goto l62
					}
					goto l33
				l62:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('a') {
						goto l66
					}
					position++
					if buffer[position] != rune('d') {
						goto l66
					}
					position++
					if buffer[position] != rune('d') {
						goto l66
					}
					position++
					if buffer[position] != rune('r') {
						goto l66
					}
					position++
					if buffer[position] != rune('e') {
						goto l66
					}
					position++
					if buffer[position] != rune('s') {
						goto l66
					}
					position++
					if buffer[position] != rune('s') {
						goto l66
					}
					position++
					if !_rules[rulespaces]() {
						goto l66
					}
					if buffer[position] != rune('a') {
						goto l66
					}
					position++
					if buffer[position] != rune('d') {
						goto l66
					}
					position++
					if buffer[position] != rune('d') {
						goto l66
					}
					position++
					if !_rules[rulespaces]() {
						goto l66
					}
					{
						position67 := position
						if !matchDot() {
							goto l66
						}
					l68:
						{
							position69, tokenIndex69 := position, tokenIndex
							if !matchDot() {
								goto l69
							}
							goto l68
						l69:
							position, tokenIndex = position69, tokenIndex69
						}
						add(rulePegText, position67)
					}
					if !_rules[ruleAction18]() {
						goto l66
					}
					if !_rules[ruleEOT]() {
						goto l66
					}
					goto l33
				l66:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('a') {
						goto l70
					}
					position++
					if buffer[position] != rune('d') {
						goto l70
					}
					position++
					if buffer[position] != rune('d') {
						goto l70
					}
					position++
					if buffer[position] != rune('r') {
						goto l70
					}
					position++
					if buffer[position] != rune('e') {
						goto l70
					}
					position++
					if buffer[position] != rune('s') {
						goto l70
					}
					position++
					if buffer[position] != rune('s') {
						goto l70
					}
					position++
					if !_rules[rulespaces]() {
						goto l70
					}
					if buffer[position] != rune('d') {
						goto l70
					}
					position++
					if buffer[position] != rune('e') {
						goto l70
					}
					position++
					if buffer[position] != rune('l') {
						goto l70
					}
					position++
					if !_rules[rulespaces]() {
						goto l70
					}
					if !_rules[rulenetwork]() {
						goto l70
					}
					if !_rules[rulespaces]() {
						goto l70
					}
					{
						position71 := position
						if !matchDot() {
							goto l70
						}
					l72:
						{
							position73, tokenIndex73 := position, tokenIndex
							if !matchDot() {
								goto l73
							}
							goto l72
						l73:
							position, tokenIndex = position73, tokenIndex73
						}
						add(rulePegText, position71)
					}
					if !_rules[ruleAction19]() {
						goto l70
					}
					if !_rules[ruleEOT]() {
						goto l70
					}
					goto l33
				l70:
					position, tokenIndex = position33, tokenIndex33
					if buffer[position] != rune('a') {
						goto l74
					}
					position++
					if buffer[position] != rune('d') {
						goto l74
					}
					position++
					if buffer[position] != rune('d') {
						goto l74
					}
					position++
					if buffer[position] != rune('r') {
						goto l74
					}
					position++
					if buffer[position] != rune('e') {
						goto l74
					}
					position++
					if buffer[position] != rune('s') {
						goto l74
					}
					position++
					if buffer[position] != rune('s') {
						goto l74
					}
					position++
					if !_rules[rulespaces]() {
						goto l74
					}
					if buffer[position] != rune('d') {
						goto l74
					}
					position++
					if buffer[position] != rune('e') {
						goto l74
					}
					position++
					if buffer[position] != rune('l') {
						goto l74
					}
					position++
					if !_rules[rulespaces]() {
						goto l74
					}
					{
						position75 := position
						if !matchDot() {
							goto l74
						}
					l76:
						{
							position77, tokenIndex77 := position, tokenIndex
							if !matchDot() {
								goto l77
							}
							goto l76
						l77:
							position, tokenIndex = position77, tokenIndex77
						}
						add(rulePegText, position75)
					}
					if !_rules[ruleAction20]() {
						goto l74
					}
					if !_rules[ruleEOT]() {
						goto l74
					}
					goto l33
				l74:
					position, tokenIndex = position33, tokenIndex33
				}
			l33:
				add(ruleoperation, position32)
			}
			return true
		},
		/* 5 network <- <((addrstr '/' len Action21) / ('d' 'e' 'f' 'a' 'u' 'l' 't' Action22))> */
		func() bool {
			position78, tokenIndex78 := position, tokenIndex
			{
				position79 := position
				{
					position80, tokenIndex80 := position, tokenIndex
					if !_rules[ruleaddrstr]() {
						goto l81
					}
					if buffer[position] != rune('/') {
						goto l81
					}
					position++
					if !_rules[rulelen]() {
						goto l81
					}
					if !_rules[ruleAction21]() {
						goto l81
					}
					goto l80
				l81:
					position, tokenIndex = position80, tokenIndex80
					if buffer[position] != rune('d') {
						goto l78
					}
					position++
					if buffer[position] != rune('e') {
						goto l78
					}
					position++
					if buffer[position] != rune('f') {
						goto l78
					}
					position++
					if buffer[position] != rune('a') {
						goto l78
					}
					position++
					if buffer[position] != rune('u') {
						goto l78
					}
					position++
					if buffer[position] != rune('l') {
						goto l78
					}
					position++
					if buffer[position] != rune('t') {
						goto l78
					}
					position++
					if !_rules[ruleAction22]() {
						goto l78
					}
				}
			l80:
				add(rulenetwork, position79)
			}
			return true
		l78:
			position, tokenIndex = position78, tokenIndex78
			return false
		},
		/* 6 addrstr <- <(<([0-9] / [a-f] / [A-F] / ':' / '.')+> Action23)> */
		func() bool {
			position82, tokenIndex82 := position, tokenIndex
			{
				position83 := position
				{
					position84 := position
					{
						position87, tokenIndex87 := position, tokenIndex
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l88
						}
						position++
						goto l87
					l88:
						position, tokenIndex = position87, tokenIndex87
						if c := buffer[position]; c < rune('a') || c > rune('f') {
							goto l89
						}
						position++
						goto l87
					l89:
						position, tokenIndex = position87, tokenIndex87
						if c := buffer[position]; c < rune('A') || c > rune('F') {
							goto l90
						}
						position++
						goto l87
					l90:
						position, tokenIndex = position87, tokenIndex87
						if buffer[position] != rune(':') {
							goto l91
						}
						position++
						goto l87
					l91:
						position, tokenIndex = position87, tokenIndex87
						if buffer[position] != rune('.') {
							goto l82
						}
						position++
					}
				l87:
				l85:
					{
						position86, tokenIndex86 := position, tokenIndex
						{
							position92, tokenIndex92 := position, tokenIndex
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l93
							}
							position++
							goto l92
						l93:
							position, tokenIndex = position92, tokenIndex92
							if c := buffer[position]; c < rune('a') || c > rune('f') {
								goto l94
							}
							position++
							goto l92
						l94:
							position, tokenIndex = position92, tokenIndex92
							if c := buffer[position]; c < rune('A') || c > rune('F') {
								goto l95
							}
							position++
							goto l92
						l95:
							position, tokenIndex = position92, tokenIndex92
							if buffer[position] != rune(':') {
								goto l96
							}
							position++
							goto l92
						l96:
							position, tokenIndex = position92, tokenIndex92
							if buffer[position] != rune('.') {
								goto l86
							}
							position++
						}
					l92:
						goto l85
					l86:
						position, tokenIndex = position86, tokenIndex86
					}
					add(rulePegText, position84)
				}
				if !_rules[ruleAction23]() {
					goto l82
				}
				add(ruleaddrstr, position83)
			}
			return true
		l82:
			position, tokenIndex = position82, tokenIndex82
			return false
		},
		/* 7 len <- <(<[0-9]+> Action24)> */
		func() bool {
			position97, tokenIndex97 := position, tokenIndex
			{
				position98 := position
				{
					position99 := position
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l97
					}
					position++
				l100:
					{
						position101, tokenIndex101 := position, tokenIndex
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l101
						}
						position++
						goto l100
					l101:
						position, tokenIndex = position101, tokenIndex101
					}
					add(rulePegText, position99)
				}
				if !_rules[ruleAction24]() {
					goto l97
				}
				add(rulelen, position98)
			}
			return true
		l97:
			position, tokenIndex = position97, tokenIndex97
			return false
		},
		/* 8 option <- <(('v' 'i' 'a' spaces <(!' ' .)+> Action25) / ('d' 'e' 'v' spaces <(!' ' .)+> Action26))> */
		func() bool {
			position102, tokenIndex102 := position, tokenIndex
			{
				position103 := position
				{
					position104, tokenIndex104 := position, tokenIndex
					if buffer[position] != rune('v') {
						goto l105
					}
					position++
					if buffer[position] != rune('i') {
						goto l105
					}
					position++
					if buffer[position] != rune('a') {
						goto l105
					}
					position++
					if !_rules[rulespaces]() {
						goto l105
					}
					{
						position106 := position
						{
							position109, tokenIndex109 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l109
							}
							position++
							goto l105
						l109:
							position, tokenIndex = position109, tokenIndex109
						}
						if !matchDot() {
							goto l105
						}
					l107:
						{
							position108, tokenIndex108 := position, tokenIndex
							{
								position110, tokenIndex110 := position, tokenIndex
								if buffer[position] != rune(' ') {
									goto l110
								}
								position++
								goto l108
							l110:
								position, tokenIndex = position110, tokenIndex110
							}
							if !matchDot() {
								goto l108
							}
							goto l107
						l108:
							position, tokenIndex = position108, tokenIndex108
						}
						add(rulePegText, position106)
					}
					if !_rules[ruleAction25]() {
						goto l105
					}
					goto l104
				l105:
					position, tokenIndex = position104, tokenIndex104
					if buffer[position] != rune('d') {
						goto l102
					}
					position++
					if buffer[position] != rune('e') {
						goto l102
					}
					position++
					if buffer[position] != rune('v') {
						goto l102
					}
					position++
					if !_rules[rulespaces]() {
						goto l102
					}
					{
						position111 := position
						{
							position114, tokenIndex114 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l114
							}
							position++
							goto l102
						l114:
							position, tokenIndex = position114, tokenIndex114
						}
						if !matchDot() {
							goto l102
						}
					l112:
						{
							position113, tokenIndex113 := position, tokenIndex
							{
								position115, tokenIndex115 := position, tokenIndex
								if buffer[position] != rune(' ') {
									goto l115
								}
								position++
								goto l113
							l115:
								position, tokenIndex = position115, tokenIndex115
							}
							if !matchDot() {
								goto l113
							}
							goto l112
						l113:
							position, tokenIndex = position113, tokenIndex113
						}
						add(rulePegText, position111)
					}
					if !_rules[ruleAction26]() {
						goto l102
					}
				}
			l104:
				add(ruleoption, position103)
			}
			return true
		l102:
			position, tokenIndex = position102, tokenIndex102
			return false
		},
		/* 9 spaces <- <(' ' / '\t')*> */
		func() bool {
			{
				position117 := position
			l118:
				{
					position119, tokenIndex119 := position, tokenIndex
					{
						position120, tokenIndex120 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l121
						}
						position++
						goto l120
					l121:
						position, tokenIndex = position120, tokenIndex120
						if buffer[position] != rune('\t') {
							goto l119
						}
						position++
					}
				l120:
					goto l118
				l119:
					position, tokenIndex = position119, tokenIndex119
				}
				add(rulespaces, position117)
			}
			return true
		},
		nil,
		/* 12 Action0 <- <{p.Err(begin, buffer, "")}> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 13 Action1 <- <{ p.TargetType = NSNONE }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 14 Action2 <- <{p.Err(begin, buffer, "")}> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 15 Action3 <- <{p.TargetType = DOCKER}> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 16 Action4 <- <{p.TargetType = NETNS}> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 17 Action5 <- <{p.TargetType = PID}> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 18 Action6 <- <{p.Err(begin, buffer, "Invalid namespace")}> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 19 Action7 <- <{p.Target = text}> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 20 Action8 <- <{p.Operation = ROUTEADD}> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 21 Action9 <- <{p.Operation = ROUTEDEL}> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
		/* 22 Action10 <- <{p.Err(begin, buffer, "Invalid option")}> */
		func() bool {
			{
				add(ruleAction10, position)
			}
			return true
		},
		/* 23 Action11 <- <{p.Err(begin, buffer, "invalid network")}> */
		func() bool {
			{
				add(ruleAction11, position)
			}
			return true
		},
		/* 24 Action12 <- <{p.Err(begin, buffer, "Invalid option")}> */
		func() bool {
			{
				add(ruleAction12, position)
			}
			return true
		},
		/* 25 Action13 <- <{p.Err(begin, buffer, "Invalid network")}> */
		func() bool {
			{
				add(ruleAction13, position)
			}
			return true
		},
		/* 26 Action14 <- <{p.Err(begin, buffer, "")}> */
		func() bool {
			{
				add(ruleAction14, position)
			}
			return true
		},
		/* 27 Action15 <- <{p.Operation = ADDRADD}> */
		func() bool {
			{
				add(ruleAction15, position)
			}
			return true
		},
		/* 28 Action16 <- <{p.Operation = ADDRDEL}> */
		func() bool {
			{
				add(ruleAction16, position)
			}
			return true
		},
		/* 29 Action17 <- <{p.Err(begin, buffer, "Invalid option")}> */
		func() bool {
			{
				add(ruleAction17, position)
			}
			return true
		},
		/* 30 Action18 <- <{p.Err(begin, buffer, "Invalid address")}> */
		func() bool {
			{
				add(ruleAction18, position)
			}
			return true
		},
		/* 31 Action19 <- <{p.Err(begin, buffer, "Invalid option")}> */
		func() bool {
			{
				add(ruleAction19, position)
			}
			return true
		},
		/* 32 Action20 <- <{p.Err(begin, buffer, "Invalid address")}> */
		func() bool {
			{
				add(ruleAction20, position)
			}
			return true
		},
		/* 33 Action21 <- <{p.IsDefault = false}> */
		func() bool {
			{
				add(ruleAction21, position)
			}
			return true
		},
		/* 34 Action22 <- <{p.IsDefault = true}> */
		func() bool {
			{
				add(ruleAction22, position)
			}
			return true
		},
		/* 35 Action23 <- <{p.Network = text}> */
		func() bool {
			{
				add(ruleAction23, position)
			}
			return true
		},
		/* 36 Action24 <- <{p.NetworkLength = text}> */
		func() bool {
			{
				add(ruleAction24, position)
			}
			return true
		},
		/* 37 Action25 <- <{p.SetOption("via", text)}> */
		func() bool {
			{
				add(ruleAction25, position)
			}
			return true
		},
		/* 38 Action26 <- <{p.SetOption("dev", text)}> */
		func() bool {
			{
				add(ruleAction26, position)
			}
			return true
		},
	}
	p.rules = _rules
}
