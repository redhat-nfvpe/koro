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
	rules  [29]func() bool
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
			p.Err(begin, buffer)
		case ruleAction1:
			p.TargetType = NSNONE
		case ruleAction2:
			p.Err(begin, buffer)
		case ruleAction3:
			p.TargetType = DOCKER
		case ruleAction4:
			p.TargetType = NETNS
		case ruleAction5:
			p.TargetType = PID
		case ruleAction6:
			p.Target = text
		case ruleAction7:
			p.Operation = ROUTEADD
		case ruleAction8:
			p.Operation = ROUTEDEL
		case ruleAction9:
			p.Operation = ADDRADD
		case ruleAction10:
			p.Operation = ADDRDEL
		case ruleAction11:
			p.IsDefault = false
		case ruleAction12:
			p.IsDefault = true
		case ruleAction13:
			p.Network = text
		case ruleAction14:
			p.NetworkLength = text
		case ruleAction15:
			p.SetOption("via", text)
		case ruleAction16:
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
		/* 0 root <- <((netns spaces operation EOT) / (netns spaces operation <.+> Action0 EOT) / (operation EOT Action1) / (<.+> Action2 EOT))> */
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
					if !_rules[ruleoperation]() {
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
					if !_rules[ruleEOT]() {
						goto l8
					}
					if !_rules[ruleAction1]() {
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
		/* 2 netns <- <(('d' 'o' 'c' 'k' 'e' 'r' spaces netnsid Action3) / ('n' 'e' 't' 'n' 's' spaces netnsid Action4) / ('p' 'i' 'd' spaces netnsid Action5))> */
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
						goto l15
					}
					position++
					if buffer[position] != rune('i') {
						goto l15
					}
					position++
					if buffer[position] != rune('d') {
						goto l15
					}
					position++
					if !_rules[rulespaces]() {
						goto l15
					}
					if !_rules[rulenetnsid]() {
						goto l15
					}
					if !_rules[ruleAction5]() {
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
		/* 3 netnsid <- <(<(!' ' .)+> Action6)> */
		func() bool {
			position20, tokenIndex20 := position, tokenIndex
			{
				position21 := position
				{
					position22 := position
					{
						position25, tokenIndex25 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l25
						}
						position++
						goto l20
					l25:
						position, tokenIndex = position25, tokenIndex25
					}
					if !matchDot() {
						goto l20
					}
				l23:
					{
						position24, tokenIndex24 := position, tokenIndex
						{
							position26, tokenIndex26 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l26
							}
							position++
							goto l24
						l26:
							position, tokenIndex = position26, tokenIndex26
						}
						if !matchDot() {
							goto l24
						}
						goto l23
					l24:
						position, tokenIndex = position24, tokenIndex24
					}
					add(rulePegText, position22)
				}
				if !_rules[ruleAction6]() {
					goto l20
				}
				add(rulenetnsid, position21)
			}
			return true
		l20:
			position, tokenIndex = position20, tokenIndex20
			return false
		},
		/* 4 operation <- <(('r' 'o' 'u' 't' 'e' spaces ('a' 'd' 'd') spaces network (spaces option)* Action7) / ('r' 'o' 'u' 't' 'e' spaces ('d' 'e' 'l') spaces network (spaces option)* Action8) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('a' 'd' 'd') spaces network spaces option Action9) / ('a' 'd' 'd' 'r' 'e' 's' 's' spaces ('d' 'e' 'l') spaces network spaces option Action10))> */
		func() bool {
			position27, tokenIndex27 := position, tokenIndex
			{
				position28 := position
				{
					position29, tokenIndex29 := position, tokenIndex
					if buffer[position] != rune('r') {
						goto l30
					}
					position++
					if buffer[position] != rune('o') {
						goto l30
					}
					position++
					if buffer[position] != rune('u') {
						goto l30
					}
					position++
					if buffer[position] != rune('t') {
						goto l30
					}
					position++
					if buffer[position] != rune('e') {
						goto l30
					}
					position++
					if !_rules[rulespaces]() {
						goto l30
					}
					if buffer[position] != rune('a') {
						goto l30
					}
					position++
					if buffer[position] != rune('d') {
						goto l30
					}
					position++
					if buffer[position] != rune('d') {
						goto l30
					}
					position++
					if !_rules[rulespaces]() {
						goto l30
					}
					if !_rules[rulenetwork]() {
						goto l30
					}
				l31:
					{
						position32, tokenIndex32 := position, tokenIndex
						if !_rules[rulespaces]() {
							goto l32
						}
						if !_rules[ruleoption]() {
							goto l32
						}
						goto l31
					l32:
						position, tokenIndex = position32, tokenIndex32
					}
					if !_rules[ruleAction7]() {
						goto l30
					}
					goto l29
				l30:
					position, tokenIndex = position29, tokenIndex29
					if buffer[position] != rune('r') {
						goto l33
					}
					position++
					if buffer[position] != rune('o') {
						goto l33
					}
					position++
					if buffer[position] != rune('u') {
						goto l33
					}
					position++
					if buffer[position] != rune('t') {
						goto l33
					}
					position++
					if buffer[position] != rune('e') {
						goto l33
					}
					position++
					if !_rules[rulespaces]() {
						goto l33
					}
					if buffer[position] != rune('d') {
						goto l33
					}
					position++
					if buffer[position] != rune('e') {
						goto l33
					}
					position++
					if buffer[position] != rune('l') {
						goto l33
					}
					position++
					if !_rules[rulespaces]() {
						goto l33
					}
					if !_rules[rulenetwork]() {
						goto l33
					}
				l34:
					{
						position35, tokenIndex35 := position, tokenIndex
						if !_rules[rulespaces]() {
							goto l35
						}
						if !_rules[ruleoption]() {
							goto l35
						}
						goto l34
					l35:
						position, tokenIndex = position35, tokenIndex35
					}
					if !_rules[ruleAction8]() {
						goto l33
					}
					goto l29
				l33:
					position, tokenIndex = position29, tokenIndex29
					if buffer[position] != rune('a') {
						goto l36
					}
					position++
					if buffer[position] != rune('d') {
						goto l36
					}
					position++
					if buffer[position] != rune('d') {
						goto l36
					}
					position++
					if buffer[position] != rune('r') {
						goto l36
					}
					position++
					if buffer[position] != rune('e') {
						goto l36
					}
					position++
					if buffer[position] != rune('s') {
						goto l36
					}
					position++
					if buffer[position] != rune('s') {
						goto l36
					}
					position++
					if !_rules[rulespaces]() {
						goto l36
					}
					if buffer[position] != rune('a') {
						goto l36
					}
					position++
					if buffer[position] != rune('d') {
						goto l36
					}
					position++
					if buffer[position] != rune('d') {
						goto l36
					}
					position++
					if !_rules[rulespaces]() {
						goto l36
					}
					if !_rules[rulenetwork]() {
						goto l36
					}
					if !_rules[rulespaces]() {
						goto l36
					}
					if !_rules[ruleoption]() {
						goto l36
					}
					if !_rules[ruleAction9]() {
						goto l36
					}
					goto l29
				l36:
					position, tokenIndex = position29, tokenIndex29
					if buffer[position] != rune('a') {
						goto l27
					}
					position++
					if buffer[position] != rune('d') {
						goto l27
					}
					position++
					if buffer[position] != rune('d') {
						goto l27
					}
					position++
					if buffer[position] != rune('r') {
						goto l27
					}
					position++
					if buffer[position] != rune('e') {
						goto l27
					}
					position++
					if buffer[position] != rune('s') {
						goto l27
					}
					position++
					if buffer[position] != rune('s') {
						goto l27
					}
					position++
					if !_rules[rulespaces]() {
						goto l27
					}
					if buffer[position] != rune('d') {
						goto l27
					}
					position++
					if buffer[position] != rune('e') {
						goto l27
					}
					position++
					if buffer[position] != rune('l') {
						goto l27
					}
					position++
					if !_rules[rulespaces]() {
						goto l27
					}
					if !_rules[rulenetwork]() {
						goto l27
					}
					if !_rules[rulespaces]() {
						goto l27
					}
					if !_rules[ruleoption]() {
						goto l27
					}
					if !_rules[ruleAction10]() {
						goto l27
					}
				}
			l29:
				add(ruleoperation, position28)
			}
			return true
		l27:
			position, tokenIndex = position27, tokenIndex27
			return false
		},
		/* 5 network <- <((addrstr '/' len) / (Action11 ('d' 'e' 'f' 'a' 'u' 'l' 't') Action12))> */
		func() bool {
			position37, tokenIndex37 := position, tokenIndex
			{
				position38 := position
				{
					position39, tokenIndex39 := position, tokenIndex
					if !_rules[ruleaddrstr]() {
						goto l40
					}
					if buffer[position] != rune('/') {
						goto l40
					}
					position++
					if !_rules[rulelen]() {
						goto l40
					}
					goto l39
				l40:
					position, tokenIndex = position39, tokenIndex39
					if !_rules[ruleAction11]() {
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
					if buffer[position] != rune('f') {
						goto l37
					}
					position++
					if buffer[position] != rune('a') {
						goto l37
					}
					position++
					if buffer[position] != rune('u') {
						goto l37
					}
					position++
					if buffer[position] != rune('l') {
						goto l37
					}
					position++
					if buffer[position] != rune('t') {
						goto l37
					}
					position++
					if !_rules[ruleAction12]() {
						goto l37
					}
				}
			l39:
				add(rulenetwork, position38)
			}
			return true
		l37:
			position, tokenIndex = position37, tokenIndex37
			return false
		},
		/* 6 addrstr <- <(<([0-9] / [a-f] / [A-F] / ':' / '.')+> Action13)> */
		func() bool {
			position41, tokenIndex41 := position, tokenIndex
			{
				position42 := position
				{
					position43 := position
					{
						position46, tokenIndex46 := position, tokenIndex
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l47
						}
						position++
						goto l46
					l47:
						position, tokenIndex = position46, tokenIndex46
						if c := buffer[position]; c < rune('a') || c > rune('f') {
							goto l48
						}
						position++
						goto l46
					l48:
						position, tokenIndex = position46, tokenIndex46
						if c := buffer[position]; c < rune('A') || c > rune('F') {
							goto l49
						}
						position++
						goto l46
					l49:
						position, tokenIndex = position46, tokenIndex46
						if buffer[position] != rune(':') {
							goto l50
						}
						position++
						goto l46
					l50:
						position, tokenIndex = position46, tokenIndex46
						if buffer[position] != rune('.') {
							goto l41
						}
						position++
					}
				l46:
				l44:
					{
						position45, tokenIndex45 := position, tokenIndex
						{
							position51, tokenIndex51 := position, tokenIndex
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l52
							}
							position++
							goto l51
						l52:
							position, tokenIndex = position51, tokenIndex51
							if c := buffer[position]; c < rune('a') || c > rune('f') {
								goto l53
							}
							position++
							goto l51
						l53:
							position, tokenIndex = position51, tokenIndex51
							if c := buffer[position]; c < rune('A') || c > rune('F') {
								goto l54
							}
							position++
							goto l51
						l54:
							position, tokenIndex = position51, tokenIndex51
							if buffer[position] != rune(':') {
								goto l55
							}
							position++
							goto l51
						l55:
							position, tokenIndex = position51, tokenIndex51
							if buffer[position] != rune('.') {
								goto l45
							}
							position++
						}
					l51:
						goto l44
					l45:
						position, tokenIndex = position45, tokenIndex45
					}
					add(rulePegText, position43)
				}
				if !_rules[ruleAction13]() {
					goto l41
				}
				add(ruleaddrstr, position42)
			}
			return true
		l41:
			position, tokenIndex = position41, tokenIndex41
			return false
		},
		/* 7 len <- <(<[0-9]+> Action14)> */
		func() bool {
			position56, tokenIndex56 := position, tokenIndex
			{
				position57 := position
				{
					position58 := position
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l56
					}
					position++
				l59:
					{
						position60, tokenIndex60 := position, tokenIndex
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l60
						}
						position++
						goto l59
					l60:
						position, tokenIndex = position60, tokenIndex60
					}
					add(rulePegText, position58)
				}
				if !_rules[ruleAction14]() {
					goto l56
				}
				add(rulelen, position57)
			}
			return true
		l56:
			position, tokenIndex = position56, tokenIndex56
			return false
		},
		/* 8 option <- <(('v' 'i' 'a' spaces <(!' ' .)+> Action15) / ('d' 'e' 'v' spaces <(!' ' .)+> Action16))> */
		func() bool {
			position61, tokenIndex61 := position, tokenIndex
			{
				position62 := position
				{
					position63, tokenIndex63 := position, tokenIndex
					if buffer[position] != rune('v') {
						goto l64
					}
					position++
					if buffer[position] != rune('i') {
						goto l64
					}
					position++
					if buffer[position] != rune('a') {
						goto l64
					}
					position++
					if !_rules[rulespaces]() {
						goto l64
					}
					{
						position65 := position
						{
							position68, tokenIndex68 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l68
							}
							position++
							goto l64
						l68:
							position, tokenIndex = position68, tokenIndex68
						}
						if !matchDot() {
							goto l64
						}
					l66:
						{
							position67, tokenIndex67 := position, tokenIndex
							{
								position69, tokenIndex69 := position, tokenIndex
								if buffer[position] != rune(' ') {
									goto l69
								}
								position++
								goto l67
							l69:
								position, tokenIndex = position69, tokenIndex69
							}
							if !matchDot() {
								goto l67
							}
							goto l66
						l67:
							position, tokenIndex = position67, tokenIndex67
						}
						add(rulePegText, position65)
					}
					if !_rules[ruleAction15]() {
						goto l64
					}
					goto l63
				l64:
					position, tokenIndex = position63, tokenIndex63
					if buffer[position] != rune('d') {
						goto l61
					}
					position++
					if buffer[position] != rune('e') {
						goto l61
					}
					position++
					if buffer[position] != rune('v') {
						goto l61
					}
					position++
					if !_rules[rulespaces]() {
						goto l61
					}
					{
						position70 := position
						{
							position73, tokenIndex73 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l73
							}
							position++
							goto l61
						l73:
							position, tokenIndex = position73, tokenIndex73
						}
						if !matchDot() {
							goto l61
						}
					l71:
						{
							position72, tokenIndex72 := position, tokenIndex
							{
								position74, tokenIndex74 := position, tokenIndex
								if buffer[position] != rune(' ') {
									goto l74
								}
								position++
								goto l72
							l74:
								position, tokenIndex = position74, tokenIndex74
							}
							if !matchDot() {
								goto l72
							}
							goto l71
						l72:
							position, tokenIndex = position72, tokenIndex72
						}
						add(rulePegText, position70)
					}
					if !_rules[ruleAction16]() {
						goto l61
					}
				}
			l63:
				add(ruleoption, position62)
			}
			return true
		l61:
			position, tokenIndex = position61, tokenIndex61
			return false
		},
		/* 9 spaces <- <(' ' / '\t')*> */
		func() bool {
			{
				position76 := position
			l77:
				{
					position78, tokenIndex78 := position, tokenIndex
					{
						position79, tokenIndex79 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l80
						}
						position++
						goto l79
					l80:
						position, tokenIndex = position79, tokenIndex79
						if buffer[position] != rune('\t') {
							goto l78
						}
						position++
					}
				l79:
					goto l77
				l78:
					position, tokenIndex = position78, tokenIndex78
				}
				add(rulespaces, position76)
			}
			return true
		},
		nil,
		/* 12 Action0 <- <{p.Err(begin, buffer)}> */
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
		/* 14 Action2 <- <{p.Err(begin, buffer)}> */
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
		/* 18 Action6 <- <{p.Target = text}> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 19 Action7 <- <{p.Operation = ROUTEADD}> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 20 Action8 <- <{p.Operation = ROUTEDEL}> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 21 Action9 <- <{p.Operation = ADDRADD}> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
		/* 22 Action10 <- <{p.Operation = ADDRDEL}> */
		func() bool {
			{
				add(ruleAction10, position)
			}
			return true
		},
		/* 23 Action11 <- <{p.IsDefault = false}> */
		func() bool {
			{
				add(ruleAction11, position)
			}
			return true
		},
		/* 24 Action12 <- <{p.IsDefault = true}> */
		func() bool {
			{
				add(ruleAction12, position)
			}
			return true
		},
		/* 25 Action13 <- <{p.Network = text}> */
		func() bool {
			{
				add(ruleAction13, position)
			}
			return true
		},
		/* 26 Action14 <- <{p.NetworkLength = text}> */
		func() bool {
			{
				add(ruleAction14, position)
			}
			return true
		},
		/* 27 Action15 <- <{p.SetOption("via", text)}> */
		func() bool {
			{
				add(ruleAction15, position)
			}
			return true
		},
		/* 28 Action16 <- <{p.SetOption("dev", text)}> */
		func() bool {
			{
				add(ruleAction16, position)
			}
			return true
		},
	}
	p.rules = _rules
}
