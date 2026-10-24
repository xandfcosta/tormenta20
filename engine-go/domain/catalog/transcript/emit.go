package transcript

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// A FORMA CANÔNICA dos arquivos de dados, e quem a escreve.
//
// Os verbetes são editados à mão e também por ferramenta — o `cmd/mintuids`
// cunha um `uid`, o `cmd/bookpages` acerta uma página. Editar a AST campo a
// campo pediria uma cirurgia por ferramenta; em vez disso a ferramenta muda o
// VALOR e o emissor reescreve o arquivo inteiro, sempre igual para o mesmo
// valor. O `TestTheTranscriptIsInCanonicalForm` prende que o que está no disco É
// essa forma, e por isso uma edição à mão fora dela reprova em vez de virar
// ruído no próximo diff de ferramenta. O conserto é o `go run ./cmd/transcriptfmt`.
//
// Comentário DENTRO de um literal não sobrevive à reescrita, e isso é a regra: o
// porquê de um verbete mora no `unmodeled`/`modeled` dele, que a ficha lê, ou na
// issue.

// maxFileLines é onde uma parte se divide em duas. Abaixo do teto de 500 do
// `TestNoNewFileGoesOverTheLineCeiling`, com folga para o cabeçalho.
const maxFileLines = 480

// goSuffix monta o nome dos arquivos de dados.
const goSuffix = ".go"

// inlineWidth é até onde um literal cabe numa linha só.
const inlineWidth = 100

// Emit devolve o conteúdo de cada arquivo de dados, pelo nome do arquivo.
//
//	files, err := transcript.Emit()
func Emit() (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, r := range layouts {
		files, err := r.emit()
		if err != nil {
			return nil, fmt.Errorf("emitir %s: %w", r.name, err)
		}
		for name, body := range files {
			out[name] = body
		}
	}
	return out, nil
}

// WriteSource reescreve os arquivos de dados em `dir` a partir dos valores
// atuais, e apaga a parte que deixou de existir.
//
//	transcript.Spells[0].BookPage = 178
//	err := transcript.WriteSource("domain/catalog/transcript")
func WriteSource(dir string) error {
	files, err := Emit()
	if err != nil {
		return err
	}
	stale, err := EmittedFilesOnDisk(dir)
	if err != nil {
		return err
	}
	for _, name := range stale {
		if _, keep := files[name]; !keep {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return err
			}
		}
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// EmittedFilesOnDisk são os arquivos de `dir` que pertencem ao emissor: o
// arquivo principal e as partes de cada recurso.
func EmittedFilesOnDisk(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		for _, r := range layouts {
			if isEmittedFileOf(e.Name(), r.file) {
				out = append(out, e.Name())
			}
		}
	}
	return out, nil
}

func isEmittedFileOf(name, base string) bool {
	if name == base+goSuffix {
		return true
	}
	rest, ok := strings.CutPrefix(name, base+"_")
	if !ok {
		return false
	}
	number, ok := strings.CutSuffix(rest, goSuffix)
	if !ok || len(number) != 2 {
		return false
	}
	_, err := strconv.Atoi(number)
	return err == nil
}

// ── os recursos ──────────────────────────────────────────────────────────────

// resourceLayout é como um recurso mora em arquivos e como ele se serializa.
type resourceLayout struct {
	name    string
	file    string
	emit    func() (map[string][]byte, error)
	marshal func() ([]byte, error)
}

// listLayout é um recurso que é uma lista de verbetes, inteira num arquivo ou
// dividida em partes (`spellsPart01` …).
func listLayout[T any](name, variable, file, doc string, whole *[]T, parts []*[]T, keyed func(T) string) resourceLayout {
	marshal := func() ([]byte, error) { return marshalPlain(*whole) }
	if keyed != nil {
		marshal = func() ([]byte, error) { return keyedByID(*whole, keyed) }
	}
	return resourceLayout{
		name:    name,
		file:    file,
		marshal: marshal,
		emit: func() (map[string][]byte, error) {
			sizes := make([]int, len(parts))
			for i, p := range parts {
				sizes[i] = len(*p)
			}
			return emitList(variable, file, doc, reflect.ValueOf(*whole), sizes)
		},
	}
}

// valueLayout é um recurso que é um valor só: um objeto ou um mapa ordenado.
func valueLayout[T any](name, variable, file, doc string, value *T, marshal func() ([]byte, error)) resourceLayout {
	return resourceLayout{
		name:    name,
		file:    file,
		marshal: marshal,
		emit: func() (map[string][]byte, error) {
			body := fmt.Sprintf("// %s são %s.\nvar %s = %s\n", variable, doc, variable,
				literal(reflect.ValueOf(*value), false))
			src, err := gofmtFile(file+goSuffix, body)
			if err != nil {
				return nil, err
			}
			return map[string][]byte{file + goSuffix: src}, nil
		},
	}
}

// emitList escreve uma lista, mantendo as partes que ela já tinha. Um verbete
// a mais entra na última parte; a parte que passa de `maxFileLines` se divide.
func emitList(variable, file, doc string, whole reflect.Value, sizes []int) (map[string][]byte, error) {
	elem := typeName(whole.Type().Elem())
	items := make([]string, whole.Len())
	for i := range items {
		items[i] = literal(whole.Index(i), true) + ","
	}
	if len(sizes) == 0 {
		sizes = []int{len(items)}
	}
	sizes = fitSizes(sizes, len(items))
	for {
		files, split, err := emitParts(variable, file, doc, elem, items, sizes)
		if err != nil {
			return nil, err
		}
		if split < 0 {
			return files, nil
		}
		half := sizes[split] / 2
		next := append([]int(nil), sizes[:split]...)
		next = append(next, half, sizes[split]-half)
		sizes = append(next, sizes[split+1:]...)
	}
}

// fitSizes ajusta as partes ao total: sobra entra na última, falta sai do fim.
func fitSizes(sizes []int, total int) []int {
	out := append([]int(nil), sizes...)
	sum := 0
	for _, s := range out {
		sum += s
	}
	for sum > total && len(out) > 0 {
		last := len(out) - 1
		take := min(out[last], sum-total)
		out[last] -= take
		sum -= take
		if out[last] == 0 && len(out) > 1 {
			out = out[:last]
		}
	}
	out[len(out)-1] += total - sum
	return out
}

// emitParts devolve os arquivos, ou o índice da primeira parte que estourou.
func emitParts(variable, file, doc, elem string, items []string, sizes []int) (map[string][]byte, int, error) {
	files := map[string][]byte{}
	if len(sizes) == 1 {
		body := fmt.Sprintf("// %s são %s.\nvar %s = []%s{\n%s\n}\n", variable, doc, variable, elem,
			strings.Join(items, "\n"))
		src, err := gofmtFile(file+goSuffix, body)
		if err != nil {
			return nil, -1, err
		}
		if lineCount(src) > maxFileLines {
			return nil, 0, nil
		}
		files[file+goSuffix] = src
		return files, -1, nil
	}
	lower := strings.ToLower(variable[:1]) + variable[1:]
	var names []string
	at := 0
	for i, size := range sizes {
		part := fmt.Sprintf("%sPart%02d", lower, i+1)
		names = append(names, part)
		body := fmt.Sprintf("// Parte %d de %d de `%s` (%s).\nvar %s = []%s{\n%s\n}\n",
			i+1, len(sizes), variable, file+goSuffix, part, elem, strings.Join(items[at:at+size], "\n"))
		at += size
		name := fmt.Sprintf("%s_%02d", file, i+1) + goSuffix
		src, err := gofmtFile(name, body)
		if err != nil {
			return nil, -1, err
		}
		if lineCount(src) > maxFileLines && size > 1 {
			return nil, i, nil
		}
		files[name] = src
	}
	body := fmt.Sprintf("// %s são %s. A lista não cabe num arquivo de 500 linhas,\n"+
		"// então mora em partes, na ordem em que o livro transcrito sempre a guardou.\n"+
		"var %s = concat(\n%s,\n)\n", variable, doc, variable, strings.Join(names, ",\n"))
	src, err := gofmtFile(file+goSuffix, body)
	if err != nil {
		return nil, -1, err
	}
	files[file+goSuffix] = src
	return files, -1, nil
}

func gofmtFile(name, body string) ([]byte, error) {
	src, err := format.Source([]byte("package transcript\n\n" + body))
	if err != nil {
		return nil, fmt.Errorf("%s não formata: %w", name, err)
	}
	return src, nil
}

func lineCount(src []byte) int { return strings.Count(string(src), "\n") }

// ── os literais ──────────────────────────────────────────────────────────────

var (
	pmCostType        = reflect.TypeFor[PMCost]()
	itemUnmodeledType = reflect.TypeFor[ItemUnmodeled]()
)

// literal é o texto Go de `v`. `elided` é o contexto em que o tipo se omite: o
// elemento de uma lista.
func literal(v reflect.Value, elided bool) string {
	t := v.Type()
	switch {
	case t == pmCostType:
		c := v.Interface().(PMCost)
		if c.Written != "" {
			return "PMCost{Written: " + strconv.Quote(c.Written) + "}"
		}
		return "PMCost{Fixed: " + strconv.Itoa(c.Fixed) + "}"
	case t == itemUnmodeledType:
		u := v.Interface().(ItemUnmodeled)
		if u.ByPart != nil {
			return "ItemUnmodeled{ByPart: " + literal(reflect.ValueOf(u.ByPart), false) + "}"
		}
		return "ItemUnmodeled{Note: " + strconv.Quote(u.Note) + "}"
	}
	switch t.Kind() {
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Pointer:
		return pointerLiteral(v)
	case reflect.Slice:
		if v.IsNil() {
			return "nil"
		}
		if isOrderedMap(t) {
			return orderedMapLiteral(v, elided)
		}
		items := make([]string, v.Len())
		for i := range items {
			items[i] = literal(v.Index(i), true)
		}
		return composite(prefix(t, elided), items)
	case reflect.Struct:
		return structLiteral(v, elided)
	}
	panic("transcript: literal sem regra para " + t.String())
}

func pointerLiteral(v reflect.Value) string {
	if v.IsNil() {
		return "nil"
	}
	inner := v.Elem()
	switch inner.Kind() {
	case reflect.Struct:
		return "&" + literal(inner, false)
	case reflect.Float64:
		return "ptr[float64](" + literal(inner, false) + ")"
	}
	return "ptr(" + literal(inner, false) + ")"
}

func orderedMapLiteral(v reflect.Value, elided bool) string {
	items := make([]string, v.Len())
	for i := range items {
		e := v.Index(i)
		items[i] = "{" + strconv.Quote(e.Field(0).String()) + ", " + literal(e.Field(1), false) + "}"
	}
	return composite(prefix(v.Type(), elided), items)
}

func structLiteral(v reflect.Value, elided bool) string {
	t := v.Type()
	var parts []string
	for i := range t.NumField() {
		f := t.Field(i)
		fv := v.Field(i)
		if !present(f, fv) {
			continue
		}
		parts = append(parts, f.Name+": "+literal(fv, false))
	}
	return composite(prefix(t, elided), parts)
}

// present diz se o campo aparece no literal: exatamente quando ele aparece no
// JSON, que é o que mantém o literal e a serialização contando a mesma história.
func present(f reflect.StructField, v reflect.Value) bool {
	_, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
	switch opts {
	case "omitempty":
		switch v.Kind() {
		case reflect.Slice, reflect.String:
			return v.Len() > 0
		case reflect.Pointer:
			return !v.IsNil()
		}
		return !v.IsZero()
	case "omitzero":
		return !v.IsZero()
	}
	return true
}

func composite(prefix string, items []string) string {
	if len(items) == 0 {
		return prefix + "{}"
	}
	one := prefix + "{" + strings.Join(items, ", ") + "}"
	if !strings.Contains(one, "\n") && len(one) <= inlineWidth {
		return one
	}
	return prefix + "{\n" + strings.Join(items, ",\n") + ",\n}"
}

func prefix(t reflect.Type, elided bool) string {
	if elided {
		return ""
	}
	return typeName(t)
}

func isOrderedMap(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && strings.HasPrefix(t.Name(), "OrderedMap[")
}

// typeName é o nome do tipo como se escreve DENTRO deste pacote.
func typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + typeName(t.Elem())
	case reflect.Slice:
		if isOrderedMap(t) {
			return "OrderedMap[" + typeName(t.Elem().Field(1).Type) + "]"
		}
		return "[]" + typeName(t.Elem())
	}
	return t.Name()
}
