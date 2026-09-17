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
	folha := compiledStylesheet(t)
	arquivos := houseSources(t)
	usadas := map[string][]string{}
	for _, caminho := range arquivos {
		fonte, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for _, tinta := range tintasDaCasaEm(semOsComentarios(string(fonte))) {
			usadas[tinta] = append(usadas[tinta], filepath.Base(caminho))
		}
	}
	// O DENOMINADOR, e ele é o que separa "nenhuma tinta reprovou" de "não varri
	// nada": um padrão que parasse de casar daria zero tintas e este guarda
	// passaria verde afirmando nada.
	//
	// O piso mede o conjunto de HOJE, e SOBE quando a varredura cresce. Deixar
	// um piso que o conjunto ANTIGO já satisfazia faria a volta acidental à
	// lista curta passar verde.
	if len(usadas) < 40 {
		t.Fatalf("a varredura achou %d tintas da casa, e são dezenas: o padrão parou de casar", len(usadas))
	}

	nomes := make([]string, 0, len(usadas))
	for nome := range usadas {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		if aFolhaConhece(folha, nome) {
			continue
		}
		t.Errorf("a tinta %q não existe na folha (usada em %s): o elemento sai com a cor herdada e ninguém reclama",
			nome, strings.Join(usadas[nome], ", "))
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
	fora := []string{}
	interessa := func(caminho string) bool {
		if strings.HasSuffix(caminho, "_test.go") || strings.HasSuffix(caminho, "_templ.go") {
			return false
		}
		return strings.HasSuffix(caminho, ".templ") || strings.HasSuffix(caminho, ".go")
	}

	// O DIRETÓRIO INTEIRO, e não um padrão de nome. O glob daqui já foi
	// `piloto_*` e deixou de casar DUAS vezes — quando o kit mudou de pasta e
	// quando os arquivos perderam o prefixo —, e nas duas o guarda seguiria
	// VERDE medindo menos. Um padrão de NOME acopla o guarda à nomenclatura; o
	// diretório é o terreno, e ele não muda de nome sozinho.
	daqui, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o diretório do api: %v", err)
	}
	for _, entrada := range daqui {
		if !entrada.IsDir() && interessa(entrada.Name()) {
			fora = append(fora, entrada.Name())
		}
	}

	// O `web/` INTEIRO, e não um pacote por linha: uma cena que caia fora da
	// lista enumerada derruba o denominador, e enumerar faria a PRÓXIMA cena
	// nascer sem medição.
	if err := filepath.WalkDir("../web", func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entrada.IsDir() && interessa(caminho) {
			fora = append(fora, caminho)
		}
		return nil
	}); err != nil {
		t.Fatalf("varrer o web/: %v", err)
	}
	if len(fora) == 0 {
		t.Fatal("nenhuma fonte do app encontrada: este guarda mediria o vazio")
	}
	return fora
}

// semOsComentarios tira as linhas de comentário antes da varredura: uma
// docstring que CITA a classe errada — como a que explica este guarda — não é
// tinta escrita em elemento nenhum, e cobrá-la faria o guarda acusar prosa.
func semOsComentarios(fonte string) string {
	linhas := strings.Split(fonte, "\n")
	fora := make([]string, 0, len(linhas))
	for _, linha := range linhas {
		if strings.HasPrefix(strings.TrimSpace(linha), "//") {
			continue
		}
		fora = append(fora, linha)
	}
	return strings.Join(fora, "\n")
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
func tintasDaCasaEm(fonte string) []string {
	fora := []string{}
	vistos := map[string]bool{}
	for _, achado := range oUtilitarioDeCor.FindAllStringSubmatch(fonte, -1) {
		nome := achado[1]
		if !daCasa(nome) || vistos[nome] {
			continue
		}
		vistos[nome] = true
		fora = append(fora, nome)
	}
	return fora
}

func daCasa(nome string) bool {
	for _, paleta := range asPaletasDaCasa {
		if nome == paleta || strings.HasPrefix(nome, paleta+"-") {
			return true
		}
	}
	return false
}

// aFolhaConhece procura o nome do token seguido de algo que NÃO continue o
// nome. Sem essa borda, `arcane` passaria por causa de `arcane-ink` — e um
// token inventado que fosse prefixo de um real nunca seria pego.
func aFolhaConhece(folha, nome string) bool {
	borda := regexp.MustCompile(regexp.QuoteMeta(nome) + `([^a-z0-9-]|$)`)
	return borda.MatchString(folha)
}
