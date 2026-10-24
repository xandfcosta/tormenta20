// Command slugifyids passa o `id` de raça e origem para a forma de slug, e
// arrasta quem aponta (ALE-404).
//
// O `uid` é a identidade (ALE-402); o `id` é o ENDEREÇO, e endereço não carrega
// rótulo. Antes desta passagem a definição de raça dizia `Humano` e a raça dizia
// `humano` para o mesmo conceito, e nada no dado dizia que eram o mesmo.
//
// O ESCOPO é uma lista explícita e não um predicado: a cunhagem de uid tentou
// predicado duas vezes e as duas erraram o conjunto em silêncio.
//
// Rodar é ato deliberado como o `genoracle`: ele reescreve o livro transcrito e
// as fichas de paridade no lugar, e o `--conferir` mostra o que faria sem
// escrever.
//
//	go run ./cmd/slugifyids --conferir
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/domain/catalog/transcript"
)

// parityFile é uma ficha de paridade lida, com o texto original para a
// reescrita preservar o ESTILO dela.
type parityFile struct {
	label    string
	path     string
	original string
	tree     *jsonNode
	swaps    int
}

func main() {
	check := flag.Bool("conferir", false, "mostra o que faria, sem escrever")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	dir, err := transcriptdir.Find()
	if err != nil {
		return err
	}
	engine := filepath.Dir(filepath.Dir(filepath.Dir(dir)))
	root := filepath.Dir(engine)

	raceIDs := make([]string, len(transcript.RaceDefinitions))
	for i, r := range transcript.RaceDefinitions {
		raceIDs[i] = r.ID
	}
	var originIDs, benefitIDs []string
	for _, o := range transcript.Origins {
		originIDs = append(originIDs, o.ID)
		for _, b := range o.Benefits {
			benefitIDs = append(benefitIDs, b.ID)
		}
	}
	raceRenames := renameMap(raceIDs)
	originRenames := renameMap(originIDs)
	benefitRenames := renameMap(benefitIDs)

	fmt.Printf("raças a renomear:      %d\n", len(raceRenames))
	fmt.Printf("origens a renomear:    %d\n", len(originRenames))
	fmt.Printf("benefícios a renomear: %d\n", len(benefitRenames))

	// `id` e `raceId` na definição de raça; `id` na origem. Os campos da FICHA
	// (`race`, `origin`) valem para as fichas de paridade.
	counts := map[string]int{}
	raceLabel := "engine-go/domain/catalog/data/race-defs.json"
	originLabel := "engine-go/domain/catalog/data/origins.json"
	counts[raceLabel] = renamer{map[string]bool{"id": true, "raceId": true}, raceRenames}.
		value(reflect.ValueOf(transcript.RaceDefinitions))
	counts[originLabel] = renamer{map[string]bool{"id": true}, merged(originRenames, benefitRenames)}.
		value(reflect.ValueOf(transcript.Origins))
	catalogSwaps := counts[raceLabel] + counts[originLabel]

	parity, err := readParity(filepath.Join(engine, "parity"), root)
	if err != nil {
		return err
	}
	sheet := renamer{map[string]bool{"race": true, "origin": true}, merged(raceRenames, originRenames)}
	total := catalogSwaps
	for _, p := range parity {
		p.swaps = sheet.tree(p.tree)
		counts[p.label] = p.swaps
		total += p.swaps
	}

	labels := make([]string, 0, len(counts))
	for l := range counts {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	for _, l := range labels {
		if counts[l] > 0 {
			fmt.Printf("  %s: %d trocas\n", l, counts[l])
		}
	}

	if check {
		fmt.Printf("\n%d trocas — nada escrito (--conferir)\n", total)
		return nil
	}
	if catalogSwaps > 0 {
		if err := transcript.WriteSource(dir); err != nil {
			return fmt.Errorf("gravar o livro transcrito em %s: %w", dir, err)
		}
	}
	for _, p := range parity {
		if p.swaps > 0 {
			if err := os.WriteFile(p.path, []byte(restyled(p)), 0o644); err != nil {
				return err
			}
		}
	}
	fmt.Printf("\n%d trocas escritas\n", total)
	return nil
}

func readParity(dir, root string) ([]*parityFile, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []*parityFile
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		tree, err := parseJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("a ficha de paridade %s não é JSON: %w", path, err)
		}
		label, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		out = append(out, &parityFile{label: filepath.ToSlash(label), path: path, original: string(raw), tree: tree})
	}
	return out, nil
}

// restyled reescreve preservando o ESTILO do arquivo — compacto ou indentado.
// Reformatar tudo num estilo só produziria um diff de milhares de linhas onde a
// mudança real são algumas centenas de valores, e ninguém revisa um diff desses.
func restyled(p *parityFile) string {
	indent := 0
	if strings.Contains(strings.TrimSpace(p.original), "\n") {
		indent = 2
	}
	text := encodeJSON(p.tree, indent)
	if strings.HasSuffix(p.original, "\n") {
		text += "\n"
	}
	return text
}
