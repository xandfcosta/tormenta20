// Command mintuids cunha o `uid` — o identificador ESTÁVEL e OPACO de cada
// verbete (ALE-402).
//
// O `id` é o nome em kebab-case, legível, e MUDA quando o nome muda; o banco
// guarda referências em dezessete colunas de nove tabelas, e renomear viraria
// migração. O `uid` é a identidade que nunca muda: `<prefixo>_<dez caracteres
// base32>`, do gerador criptográfico. O prefixo diz a ESPÉCIE só para orientar
// quem lê um diff — ele não deriva do nome, porque um uid derivável não é
// estável: bastaria alguém "corrigir" a derivação para ele mudar. Dez caracteres
// base32 são 50 bits; mesmo assim a cunhagem RECUSA colidir em vez de confiar na
// conta.
//
// **Não toca um `uid` que já existe** — cunhar é operação de nascimento, e o
// `TestNoUidEverChanges` prende isso. E o uid é por CONCEITO, não por linha: o
// `divine-powers` tem 80 linhas para 72 poderes, porque o mesmo poder aparece
// uma vez por deus que o concede, e as linhas de um poder dividem um uid.
//
//	go run ./cmd/mintuids            # relatório
//	go run ./cmd/mintuids --aplicar  # cunha e reescreve o livro transcrito
package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/domain/catalog/transcript"
)

// alphabet é base32 sem 0/1/8/9, que se confundem.
const (
	alphabet = "abcdefghijklmnopqrstuvwxyz234567"
	uidSize  = 10
)

func main() {
	apply := flag.Bool("aplicar", false, "cunha e grava; sem a flag, só relata")
	flag.Parse()

	// O origins-source herda o uid da origem, e no relatório o Python lia a
	// origem do DISCO — sem o que a simulação acabou de cunhar. A foto antes da
	// cunhagem é esse disco.
	originsBefore := originUIDs()

	used := map[string]bool{}
	for _, s := range scope {
		for _, path := range s.paths() {
			for _, e := range path {
				if *e.uid != "" {
					used[*e.uid] = true
				}
			}
		}
	}

	type count struct{ lines, concepts int }
	perFile := map[string]count{}
	minted, existing, changed := 0, 0, false
	var undeclared []string
	for _, resource := range sortedFiles() {
		if outOfScope[resource] {
			continue
		}
		s, ok := scopeOf(resource)
		if !ok {
			undeclared = append(undeclared, resource)
			continue
		}
		byIdentity := map[string]string{}
		lines := 0
		for _, path := range s.paths() {
			for _, e := range path {
				lines++
				if *e.uid != "" {
					if _, seen := byIdentity[e.identity]; !seen {
						byIdentity[e.identity] = *e.uid
					}
					existing++
					continue
				}
				if _, seen := byIdentity[e.identity]; !seen {
					byIdentity[e.identity] = mint(used, s.prefix)
					minted++
				}
				*e.uid = byIdentity[e.identity]
				changed = true
			}
		}
		perFile[resource] = count{lines, len(byIdentity)}
	}

	// A HERANÇA entre arquivos que descrevem o mesmo conceito. Hoje é um caso
	// só, e ele é declarado em vez de deduzido: "mesmo nome, mesmo uid" como
	// regra geral juntaria a perícia Cura com o efeito Cura.
	source := originsBefore
	if *apply {
		source = originUIDs()
	}
	inherited := mirrorOrigins(source)

	if *apply && (changed || inherited > 0) {
		dir, err := transcriptdir.Find()
		if err == nil {
			err = transcript.WriteSource(dir)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "gravar o livro transcrito:", err)
			os.Exit(1)
		}
	}

	names := make([]string, 0, len(perFile))
	for n := range perFile {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		c := perFile[n]
		extra := ""
		if c.lines != c.concepts {
			extra = fmt.Sprintf("  (%d linhas dividem uid)", c.lines-c.concepts)
		}
		fmt.Printf("  %4d conceitos em %4d linhas  %s%s\n", c.concepts, c.lines, n, extra)
	}
	mode := "simulação (use --aplicar)"
	if *apply {
		mode = "GRAVADO"
	}
	fmt.Printf("\nuids herdados por origins-source: %d\n", inherited)
	fmt.Printf("uids já existentes: %d | cunhados agora: %d | %s\n", existing, minted, mode)
	// A ausência é DITA: recurso que ninguém declarou nem excluiu sai nomeado,
	// em vez de simplesmente não ser cunhado.
	for _, n := range undeclared {
		fmt.Printf("  NÃO DECLARADO %s: nem em ESCOPO nem em FORA — decida e volte\n", n)
	}
	if len(undeclared) > 0 {
		os.Exit(1)
	}
}

// sortedFiles são os recursos do livro transcrito pelo nome de arquivo que
// tinham — `origins-source.json` antes de `origins.json`, como o relatório
// sempre os listou.
func sortedFiles() []string {
	out := make([]string, len(transcript.Resources))
	for i, r := range transcript.Resources {
		out[i] = r + ".json"
	}
	sort.Strings(out)
	return out
}

// mint sorteia um uid novo da espécie, e sorteia de novo se ele já existir.
func mint(used map[string]bool, prefix string) string {
	buf := make([]byte, uidSize)
	for {
		if _, err := rand.Read(buf); err != nil {
			panic("crypto/rand: " + err.Error())
		}
		var b strings.Builder
		b.WriteString(prefix + "_")
		for _, x := range buf {
			// 256 é múltiplo de 32: o resto não enviesa a letra.
			b.WriteByte(alphabet[int(x)%len(alphabet)])
		}
		if uid := b.String(); !used[uid] {
			used[uid] = true
			return uid
		}
	}
}

// originUIDs é o uid de cada origem pela chave kebab do id; a última de uma
// chave repetida vence.
func originUIDs() map[string]string {
	out := map[string]string{}
	for _, o := range transcript.Origins {
		if o.UID != "" {
			out[kebabKey(o.ID)] = o.UID
		}
	}
	return out
}

// mirrorOrigins faz o `origins-source` herdar o uid da origem correspondente:
// dois recursos, um conceito — duas identidades para a mesma origem seria o
// defeito que o uid existe para impedir.
func mirrorOrigins(byKey map[string]string) int {
	n := 0
	for i := range transcript.OriginSources {
		src := &transcript.OriginSources[i]
		uid, ok := byKey[src.ID]
		if !ok || src.UID == uid {
			continue
		}
		src.UID = uid
		n++
	}
	return n
}

var notKebab = regexp.MustCompile(`[^a-z0-9]+`)

func kebabKey(name string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return strings.Trim(notKebab.ReplaceAllString(b.String(), "-"), "-")
}
