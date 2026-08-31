package api

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TINTA QUE NÃO EXISTE NA FOLHA NÃO DESENHA — e não reclama (ALE-272, fatia 6).
//
// O `text-grimorio-ink` foi escrito na fatia 5 e nunca existiu: `grimorio-ink`
// não é token da paleta. O Tailwind não emite regra para uma classe que não
// conhece, o elemento fica com a cor HERDADA, e o crachá de contagem dos
// Efeitos saiu com número dourado sobre fundo dourado — 1,53:1, ilegível. Nada
// falha: o HTML tem a classe, o `templ generate` passa, o `go build` passa, e a
// folha simplesmente não tem a regra.
//
// O medidor de contraste do e2e pegou este caso, mas por sorte de ESTADO: o
// crachá só aparece quando há efeito ativo, e no dia em que a suíte rodou
// inteira o personagem medido não tinha nenhum. Guarda que depende de o dado
// certo estar no banco é guarda que mede quando quer.
//
// Ele varre a FONTE inteira — os `.templ` e os `.go` do piloto —, e não uma
// cena servida: a varredura é o que faz a convenção valer para a próxima tela
// também.
func TestTodaTintaDaCasaExisteNaFolha(t *testing.T) {
	folha, err := os.ReadFile(filepath.Join("piloto", "static", "piloto.css"))
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	arquivos := osFontesDoPiloto(t)
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
	// O DENOMINADOR: um padrão que parasse de casar daria zero tintas e este
	// guarda passaria verde afirmando nada.
	if len(usadas) < 10 {
		t.Fatalf("a varredura achou %d tintas da casa, e são dezenas: o padrão parou de casar", len(usadas))
	}

	nomes := make([]string, 0, len(usadas))
	for nome := range usadas {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		if aFolhaConhece(string(folha), nome) {
			continue
		}
		t.Errorf("a tinta %q não existe na folha (usada em %s): o elemento sai com a cor herdada e ninguém reclama",
			nome, strings.Join(usadas[nome], ", "))
	}
}

// osFontesDoPiloto são os arquivos que escrevem classe: os `.templ`, que é o
// que o scanner do Tailwind lê, e os `.go` do piloto, onde uma classe escrita
// não passa pelo scanner e só existe se alguém a registrou no `@source inline`.
// O gerado (`_templ.go`) fica de fora porque repete o `.templ` ao lado.
func osFontesDoPiloto(t *testing.T) []string {
	t.Helper()
	fora := []string{}
	for _, padrao := range []string{"piloto_*.templ", "piloto_*.go"} {
		achados, err := filepath.Glob(padrao)
		if err != nil {
			t.Fatalf("varrer %s: %v", padrao, err)
		}
		for _, caminho := range achados {
			if strings.HasSuffix(caminho, "_test.go") || strings.HasSuffix(caminho, "_templ.go") {
				continue
			}
			fora = append(fora, caminho)
		}
	}
	if len(fora) == 0 {
		t.Fatal("nenhuma fonte do piloto encontrada: este guarda mediria o vazio")
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
var asPaletasDaCasa = []string{"grimorio", "arcane", "penalty", "hp", "mp", "terreno"}

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
