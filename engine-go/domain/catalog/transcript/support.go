// Package transcript é o livro de Tormenta 20 transcrito em Go: cada verbete do
// catálogo é um literal destes tipos, e é aqui que uma regra transcrita se edita.
//
// Até a ALE-498 ele era 24 arquivos JSON em `domain/catalog/data`. Virou Go para
// o COMPILADOR conferir o que o JSON deixava passar calado: uma chave escrita
// errado era ignorada pelo leitor, e um campo de tipo trocado só estourava em
// quem o lia. Os leitores continuam recebendo JSON — o `ResourceJSON` serializa
// estes valores —, e por isso nenhum deles mudou junto.
//
//	raw, ok := transcript.ResourceJSON("spells")
package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Entry é um par de um OrderedMap.
type Entry[V any] struct {
	Key   string
	Value V
}

// OrderedMap é um objeto JSON cujas chaves são DADO (um atributo, um círculo,
// um nome) e cuja ordem o livro fixa. O mapa de Go não guarda ordem, e há leitor
// que a usa: o `orderedInts` do motor tira os modificadores de atributo da raça
// na ordem em que o verbete os escreve.
//
//	OrderedMap[int]{{"constitution", 2}, {"wisdom", 1}}
type OrderedMap[V any] []Entry[V]

// MarshalJSON escreve o objeto na ordem das entradas.
func (m OrderedMap[V]) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, e := range m {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := marshalPlain(e.Key)
		if err != nil {
			return nil, err
		}
		value, err := marshalPlain(e.Value)
		if err != nil {
			return nil, fmt.Errorf("OrderedMap[%q]: %w", e.Key, err)
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Get devolve o valor da chave, e se ela existe.
func (m OrderedMap[V]) Get(key string) (V, bool) {
	for _, e := range m {
		if e.Key == key {
			return e.Value, true
		}
	}
	var zero V
	return zero, false
}

// PMCost é o custo em PM de uma ativação: um número, ou o texto do livro quando
// ele não é um número só ("varia", "1 por dado").
type PMCost struct {
	Fixed   int
	Written string
}

// MarshalJSON escreve o texto quando há texto, e o número quando não há.
func (c PMCost) MarshalJSON() ([]byte, error) {
	if c.Written != "" {
		return marshalPlain(c.Written)
	}
	return marshalPlain(c.Fixed)
}

// ItemUnmodeled é a nota do que um item faz e a ficha não soma: uma frase, ou
// uma frase por PARTE quando o item serve a mais de um equipamento.
type ItemUnmodeled struct {
	Note   string
	ByPart OrderedMap[string]
}

// MarshalJSON escreve o objeto por parte quando há partes, e a frase quando não.
func (u ItemUnmodeled) MarshalJSON() ([]byte, error) {
	if u.ByPart != nil {
		return u.ByPart.MarshalJSON()
	}
	return marshalPlain(u.Note)
}

// ptr é o endereço de um literal, para os campos que distinguem "ausente" de
// "zero" ou de "nulo".
func ptr[T any](v T) *T { return &v }

// concat junta as partes de uma lista que não cabe num arquivo de 500 linhas.
func concat[T any](parts ...[]T) []T {
	var out []T
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// marshalPlain serializa sem escapar `<`, `>` e `&`: o texto do livro tem os
// três, e o escape só mudaria bytes que o leitor desfaz.
func marshalPlain[T any](v T) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// keyedByID escreve uma lista como objeto por id, na ordem da lista: é a forma
// em que as magias, as raças, as condições, as fontes de origem e os poderes da
// Tormenta sempre foram servidos.
func keyedByID[T any](list []T, id func(T) string) ([]byte, error) {
	m := make(OrderedMap[T], len(list))
	for i, v := range list {
		m[i] = Entry[T]{Key: id(v), Value: v}
	}
	return m.MarshalJSON()
}
