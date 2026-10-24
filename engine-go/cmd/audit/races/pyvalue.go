package main

import (
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
)

// pyDict é o `atributoMod` como o Python o via: um dicionário cuja IGUALDADE
// ignora a ordem e cujo `repr` a preserva. A comparação do auditor é de
// dicionário com dicionário, e a mensagem imprime os dois como o Python
// imprimia.
type pyDict []pyEntry

type pyEntry struct {
	key   string
	value pyValue
}

// pyValue é um escalar já no `repr` dele, ou um dicionário.
type pyValue struct {
	scalar string
	dict   pyDict
	isDict bool
}

func pyInt(n int) pyValue       { return pyValue{scalar: strconv.Itoa(n)} }
func pyText(s string) pyValue   { return pyValue{scalar: pyfmt.Repr(s)} }
func pyNested(d pyDict) pyValue { return pyValue{dict: d, isDict: true} }

// set grava a chave mantendo a posição de quem já estava, como o `dict`.
func (d *pyDict) set(key string, v pyValue) {
	for i := range *d {
		if (*d)[i].key == key {
			(*d)[i].value = v
			return
		}
	}
	*d = append(*d, pyEntry{key, v})
}

func (d pyDict) get(key string) (pyValue, bool) {
	for _, e := range d {
		if e.key == key {
			return e.value, true
		}
	}
	return pyValue{}, false
}

func (d pyDict) equal(o pyDict) bool {
	if len(d) != len(o) {
		return false
	}
	for _, e := range d {
		v, ok := o.get(e.key)
		if !ok || !e.value.equal(v) {
			return false
		}
	}
	return true
}

func (v pyValue) equal(o pyValue) bool {
	if v.isDict != o.isDict {
		return false
	}
	if v.isDict {
		return v.dict.equal(o.dict)
	}
	return v.scalar == o.scalar
}

func (v pyValue) repr() string {
	if v.isDict {
		return v.dict.repr()
	}
	return v.scalar
}

func (d pyDict) repr() string {
	parts := make([]string, len(d))
	for i, e := range d {
		parts[i] = pyfmt.Repr(e.key) + ": " + e.value.repr()
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// ruleDict é o `atributoMod` do catálogo com as chaves que o JSON dele trazia:
// as mesmas regras de `omitempty` do tipo, na ordem dos campos.
func ruleDict(r transcript.RaceAttributeRule) pyDict {
	d := pyDict{{"kind", pyText(r.Kind)}}
	if r.Count != 0 {
		d.set("count", pyInt(r.Count))
	}
	if r.Value != 0 {
		d.set("value", pyInt(r.Value))
	}
	if len(r.Mods) > 0 {
		d.set("mods", pyNested(intDict(r.Mods)))
	}
	if r.Exclude != "" {
		d.set("exclude", pyText(r.Exclude))
	}
	if r.Penalty != nil {
		d.set("penalty", pyNested(pyDict{{"attribute", pyText(r.Penalty.Attribute)}, {"value", pyInt(r.Penalty.Value)}}))
	}
	if len(r.Variants) > 0 {
		var variants pyDict
		for _, e := range r.Variants {
			variants.set(e.Key, pyNested(intDict(e.Value)))
		}
		d.set("variants", pyNested(variants))
	}
	return d
}

func intDict(m transcript.OrderedMap[int]) pyDict {
	var d pyDict
	for _, e := range m {
		d.set(e.Key, pyInt(e.Value))
	}
	return d
}
