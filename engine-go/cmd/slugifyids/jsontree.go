package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
)

// jsonNode é um valor JSON que GUARDA a ordem das chaves. Os arquivos de
// paridade continuam JSON e são reescritos como o Python os reescrevia; um
// `map[string]any` embaralharia as chaves e o diff de uma troca de valor viraria
// o arquivo inteiro.
type jsonNode struct {
	kind  jsonKind
	keys  []string    // objeto
	elems []*jsonNode // valores do objeto, ou itens da lista
	text  string      // string, ou o literal de número/bool/null
}

type jsonKind int

const (
	jsonObject jsonKind = iota
	jsonArray
	jsonString
	jsonLiteral
)

// parseJSON lê um documento inteiro; sobra depois do valor é erro, como no
// `json.loads`.
func parseJSON(raw []byte) (*jsonNode, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	node, err := parseNode(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("sobra texto depois do valor JSON (esperado o fim do documento)")
	}
	return node, nil
}

func parseNode(dec *json.Decoder) (*jsonNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		node := &jsonNode{kind: jsonArray}
		if t == '{' {
			node.kind = jsonObject
		}
		for dec.More() {
			if node.kind == jsonObject {
				key, err := dec.Token()
				if err != nil {
					return nil, err
				}
				node.keys = append(node.keys, key.(string))
			}
			child, err := parseNode(dec)
			if err != nil {
				return nil, err
			}
			node.elems = append(node.elems, child)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return node, nil
	case string:
		return &jsonNode{kind: jsonString, text: t}, nil
	case json.Number:
		return &jsonNode{kind: jsonLiteral, text: pyNumber(t.String())}, nil
	case bool:
		return &jsonNode{kind: jsonLiteral, text: strconv.FormatBool(t)}, nil
	case nil:
		return &jsonNode{kind: jsonLiteral, text: "null"}, nil
	}
	return nil, fmt.Errorf("token JSON inesperado %v", tok)
}

// pyNumber é como o número volta do `json.loads` + `json.dumps`: com fração ou
// expoente ele é float e sai no `repr()` do float; sem, é int e sai canônico.
func pyNumber(lit string) string {
	if !strings.ContainsAny(lit, ".eE") {
		n, ok := new(big.Int).SetString(lit, 10)
		if !ok {
			return lit
		}
		return n.String()
	}
	f, err := strconv.ParseFloat(lit, 64)
	if err != nil {
		return lit
	}
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	mantissa, expText, _ := strings.Cut(sci, "e")
	exp, _ := strconv.Atoi(expText)
	if exp < -4 || exp >= 16 {
		sign := "+"
		if exp < 0 {
			sign, exp = "-", -exp
		}
		return fmt.Sprintf("%se%s%02d", mantissa, sign, exp)
	}
	fixed := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(fixed, ".") {
		fixed += ".0"
	}
	return fixed
}

// encodeJSON é o `json.dumps(..., ensure_ascii=False)`: compacto com
// separadores `,`/`:` quando indent é 0, e indentado com `, `→`,` e `: ` quando
// não.
func encodeJSON(n *jsonNode, indent int) string {
	var b strings.Builder
	writeNode(&b, n, indent, 0)
	return b.String()
}

func writeNode(b *strings.Builder, n *jsonNode, indent, depth int) {
	switch n.kind {
	case jsonString:
		writePyString(b, n.text)
		return
	case jsonLiteral:
		b.WriteString(n.text)
		return
	}
	open, closing := "[", "]"
	if n.kind == jsonObject {
		open, closing = "{", "}"
	}
	if len(n.elems) == 0 {
		b.WriteString(open + closing)
		return
	}
	b.WriteString(open)
	keySep := ":"
	if indent > 0 {
		keySep = ": "
	}
	for i, child := range n.elems {
		if i > 0 {
			b.WriteByte(',')
		}
		if indent > 0 {
			b.WriteString("\n" + strings.Repeat(" ", indent*(depth+1)))
		}
		if n.kind == jsonObject {
			writePyString(b, n.keys[i])
			b.WriteString(keySep)
		}
		writeNode(b, child, indent, depth+1)
	}
	if indent > 0 {
		b.WriteString("\n" + strings.Repeat(" ", indent*depth))
	}
	b.WriteString(closing)
}

// writePyString escapa só o que o `json.dumps` com `ensure_ascii=False`
// escapa: aspa, barra invertida e os controles abaixo de 0x20.
func writePyString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
