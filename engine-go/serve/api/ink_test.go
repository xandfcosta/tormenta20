package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TINTA QUE NÃO EXISTE NA FOLHA NÃO DESENHA — e não reclama.
//
// O Tailwind não emite regra para uma classe que não conhece, o elemento fica
// com a cor HERDADA, e o número sai dourado sobre fundo dourado. Nada falha: o
// HTML tem a classe, o `templ generate` passa, o `go build` passa, e a folha
// simplesmente não tem a regra.
//
// O medidor de contraste do e2e alcança alguns desses casos, mas por sorte de
// ESTADO — o crachá só aparece quando há efeito ativo. Guarda que depende de o
// dado certo estar no banco é guarda que mede quando quer.
//
// Este varre a FONTE inteira — os `.templ` e os `.go` das cenas —, e não uma
// cena servida: a varredura é o que faz a convenção valer para a próxima tela
// também.
func TestEveryHouseTintExistsInTheStylesheet(t *testing.T) {
	sheet := compiledStylesheet(t)
	files := houseSources(t)
	used := map[string][]string{}
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		for _, paint := range tintasDaCasaEm(semOsComentarios(string(source))) {
			used[paint] = append(used[paint], filepath.Base(path))
		}
	}
	// O DENOMINADOR, e ele é o que separa "nenhuma tinta reprovou" de "não varri
	// nada": um padrão que parasse de casar daria zero tintas e este guarda
	// passaria verde afirmando nada.
	//
	// O piso mede o conjunto de HOJE, e SOBE quando a varredura cresce. Deixar
	// um piso que o conjunto ANTIGO já satisfazia faria a volta acidental à
	// lista curta passar verde.
	if len(used) < 40 {
		t.Fatalf("a varredura achou %d tintas da casa, e são dezenas: o padrão parou de casar", len(used))
	}

	names := make([]string, 0, len(used))
	for name := range used {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if aFolhaConhece(sheet, name) {
			continue
		}
		t.Errorf("a tinta %q não existe na folha (usada em %s): o elemento sai com a cor herdada e ninguém reclama",
			name, strings.Join(used[name], ", "))
	}
}

// houseSources são os arquivos que escrevem classe: os `.templ`, que é o que o
// scanner do Tailwind lê, e os `.go` do app, onde uma classe escrita não passa
// pelo scanner e só existe se alguém a registrou no `@source inline`. O gerado
// (`_templ.go`) fica de fora porque repete o `.templ` ao lado.
//
// O KIT entra junto com as cenas, e não é detalhe: o botão, o campo e a casca
// são justamente os arquivos onde uma tinta errada aparece em TODA tela.
func houseSources(t *testing.T) []string {
	t.Helper()
	outside := []string{}
	matters := func(path string) bool {
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_templ.go") {
			return false
		}
		return strings.HasSuffix(path, ".templ") || strings.HasSuffix(path, ".go")
	}

	// O DIRETÓRIO INTEIRO, e não um padrão de nome. O glob daqui já foi
	// `piloto_*` e deixou de casar DUAS vezes — quando o kit mudou de pasta e
	// quando os arquivos perderam o prefixo —, e nas duas o guarda seguiria
	// VERDE medindo menos. Um padrão de NOME acopla o guarda à nomenclatura; o
	// diretório é o terreno, e ele não muda de nome sozinho.
	fromHere, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o diretório do api: %v", err)
	}
	for _, entry := range fromHere {
		if !entry.IsDir() && matters(entry.Name()) {
			outside = append(outside, entry.Name())
		}
	}

	// O `web/` INTEIRO, e não um pacote por linha: uma cena que caia fora da
	// lista enumerada derruba o denominador, e enumerar faria a PRÓXIMA cena
	// nascer sem medição.
	if err := filepath.WalkDir("../web", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && matters(path) {
			outside = append(outside, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("varrer o web/: %v", err)
	}
	if len(outside) == 0 {
		t.Fatal("nenhuma fonte do app encontrada: este guarda mediria o vazio")
	}
	return outside
}

// semOsComentarios tira as linhas de comentário antes da varredura: uma
// docstring que CITA a classe errada — como a que explica este guarda — não é
// tinta escrita em elemento nenhum, e cobrá-la faria o guarda acusar prosa.
func semOsComentarios(source string) string {
	rows := strings.Split(source, "\n")
	outside := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.HasPrefix(strings.TrimSpace(row), "//") {
			continue
		}
		outside = append(outside, row)
	}
	return strings.Join(outside, "\n")
}

// asPaletasDaCasa são os prefixos de token DESTE projeto. A paleta embutida do
// Tailwind fica de fora de propósito: ela existe sempre, e incluí-la só traria
// o ruído das variantes sem prender defeito nenhum.
//
// A FAMÍLIA SEMÂNTICA (`primary`, `muted`, `card`, `popover`, `destructive`…)
// está aqui porque a ausência dela é um buraco em silêncio: um
// `text-destructive-foreground` que nunca existiu na paleta passa por este
// guarda inteiro se `destructive` não estiver na lista.
//
// A lista é a dos `--color-*` que o `@theme` do `index.css` declara, menos o
// que a paleta embutida do Tailwind já cobre. Token semântico novo entra aqui
// junto com a linha do `@theme` — as duas metades do mesmo ato.
var asPaletasDaCasa = []string{
	"grimorio", "arcane", "penalty", "hp", "mp", "terreno",
	"accent", "background", "bonus", "border", "card", "destructive",
	"foreground", "input", "marker", "muted", "popover", "primary",
	"ring", "secondary", "warning",
}

var oUtilitarioDeCor = regexp.MustCompile(
	`\b(?:text|bg|border|ring|outline|fill|stroke|decoration|shadow|from|via|to)-([a-z]+(?:-[a-z0-9]+)*)`)

// tintasDaCasaEm devolve os NOMES de token da casa citados no arquivo.
//
// O nome, e não a classe inteira, porque a folha escreve o mesmo token de
// várias formas — `.text-arcane-ink`, `.text-arcane-ink\/80`, `--arcane-ink` —
// e cobrar uma forma específica cobraria a implementação do Tailwind.
func tintasDaCasaEm(source string) []string {
	outside := []string{}
	seen := map[string]bool{}
	for _, found := range oUtilitarioDeCor.FindAllStringSubmatch(source, -1) {
		name := found[1]
		if !daCasa(name) || seen[name] {
			continue
		}
		seen[name] = true
		outside = append(outside, name)
	}
	return outside
}

func daCasa(name string) bool {
	for _, palette := range asPaletasDaCasa {
		if name == palette || strings.HasPrefix(name, palette+"-") {
			return true
		}
	}
	return false
}

// aFolhaConhece procura o nome do token seguido de algo que NÃO continue o
// nome. Sem essa borda, `arcane` passaria por causa de `arcane-ink` — e um
// token inventado que fosse prefixo de um real nunca seria pego.
func aFolhaConhece(sheet, name string) bool {
	border := regexp.MustCompile(regexp.QuoteMeta(name) + `([^a-z0-9-]|$)`)
	return border.MatchString(sheet)
}
