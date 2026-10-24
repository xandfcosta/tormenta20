package main

import (
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"t20engine/infra/bookpdf"
)

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug é a forma de endereço: `Herói Camponês` → `heroi-campones`.
func slug(text string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(text) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return strings.Trim(notSlug.ReplaceAllString(strings.ToLower(b.String()), "-"), "-")
}

// renameMap vai de `Humano` para `humano`, só para os ids que ainda não são
// slug.
func renameMap(ids []string) map[string]string {
	out := map[string]string{}
	for _, id := range ids {
		if s := slug(id); s != id {
			out[id] = s
		}
	}
	return out
}

func merged(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// renamer troca o VALOR dos campos nomeados, onde quer que eles estejam. Os
// campos são nomeados porque um `id` de benefício e um `raceId` são coisas
// diferentes que podem ter o mesmo texto; trocar por valor arrastaria junto uma
// descrição que por acaso dissesse `Humano`.
type renamer struct {
	fields  map[string]bool
	renames map[string]string
}

// field é a decisão por campo, igual para o literal tipado e para o JSON: o
// campo nomeado com valor a renomear troca; qualquer outro texto pode ser JSON
// embutido. Devolve o novo texto e quantas trocas houve.
func (r renamer) field(name, value string) (string, int) {
	if r.fields[name] {
		if to, ok := r.renames[value]; ok {
			return to, 1
		}
	}
	return r.embedded(value)
}

// embedded troca dentro do JSON guardado DENTRO de uma string: a ficha guarda
// escolha assim, e o ponteiro lá dentro não se parece com ponteiro —
// `secondaryRaceChoices` é `"[{\"race\":\"Lefou\",…}]"`, e a varredura que só
// olhava valores de campo passou por cima dela (o Kharvos perdeu o −1 de
// Carisma da Deformidade).
func (r renamer) embedded(text string) (string, int) {
	head := bookpdf.Strip(text)
	if head != "" && head[0] != '[' && head[0] != '{' {
		return text, 0
	}
	node, err := parseJSON([]byte(text))
	if err != nil {
		return text, 0
	}
	n := r.tree(node)
	if n == 0 {
		return text, 0
	}
	return encodeJSON(node, 0), n
}

// tree troca num JSON lido. Texto solto numa lista não é campo, e não é olhado.
func (r renamer) tree(n *jsonNode) int {
	count := 0
	for i, child := range n.elems {
		if n.kind == jsonObject && child.kind == jsonString {
			text, k := r.field(n.keys[i], child.text)
			child.text = text
			count += k
			continue
		}
		count += r.tree(child)
	}
	return count
}

// value troca num literal tipado do livro transcrito, campo a campo pela tag
// JSON — que é o nome com que o campo existia no arquivo de dados.
func (r renamer) value(v reflect.Value) int {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return 0
		}
		return r.value(v.Elem())
	case reflect.Slice, reflect.Array:
		count := 0
		for i := 0; i < v.Len(); i++ {
			count += r.value(v.Index(i))
		}
		return count
	case reflect.Struct:
		return r.structFields(v)
	}
	return 0
}

func (r renamer) structFields(v reflect.Value) int {
	t := v.Type()
	// Um par de OrderedMap é uma chave do objeto: a chave é o NOME do campo, e
	// só o valor se troca.
	if strings.HasPrefix(t.Name(), "Entry[") {
		key := v.FieldByName("Key").String()
		return r.member(key, v.FieldByName("Value"))
	}
	count := 0
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		count += r.member(name, v.Field(i))
	}
	return count
}

func (r renamer) member(name string, v reflect.Value) int {
	if v.Kind() == reflect.Pointer && !v.IsNil() && v.Elem().Kind() == reflect.String {
		v = v.Elem()
	}
	if v.Kind() != reflect.String {
		return r.value(v)
	}
	text, n := r.field(name, v.String())
	if n > 0 {
		v.SetString(text)
	}
	return n
}
