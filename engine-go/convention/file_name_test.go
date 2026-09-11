package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// NOME DE ARQUIVO É INGLÊS, E A DESCRIÇÃO DE DENTRO DO SPEC É PORTUGUÊS (ALE-301).
//
// O par é o que confunde, e por isso ele é decisão registrada no `CLAUDE.md` em
// vez de convenção implícita: o mesmo arquivo leva as duas línguas —
// `piloto-board.spec.ts` contendo `test('arrastar a peça propõe a parada')`. O
// arquivo é identificador, a descrição é frase que uma pessoa lê no relatório.
//
// Este guarda cobra SÓ a metade do arquivo. A descrição não tem guarda e não
// deve ter: "está em português" não é mecanizável sem um dicionário, e um
// medidor que chutasse ali reprovaria a frase certa — foi o que aconteceu na
// ALE-300, quando contar partículas ambíguas acusou trezentos nomes de teste em
// inglês correto.
//
// # A raiz é o REPOSITÓRIO, e a lista é de PERMITIDOS
//
// A fatia 1 mediu só o `e2e/tests` e disse no cabeçalho que o recorte era
// temporário. A fatia 2 varreu o resto e a raiz subiu — mas o que mudou junto,
// e importa mais, foi o INSTRUMENTO.
//
// O guarda da fatia 1 casava contra uma lista de palavras PORTUGUESAS. Ela não
// conhecia `fronteira`, e por isso a primeira contagem do `engine-go` deu 13; com
// a lista ampliada deu 18; o terreno real era **96**, dos quais 25 eram um
// guarda de fronteira por pacote — invisíveis nas três contagens, e cada uma
// delas com cara de resposta. Lista de proibidos SUBCONTA em silêncio, e
// diferente de uma decomposição ela não tem denominador embutido para denunciar.
//
// Hoje o guarda FALHA no segmento que não conhece, que é a forma da ALE-294: o
// parser que ignorava o seletor desconhecido produzia lista de falhas com cara
// de descoberta. Palavra inglesa nova custa uma linha em
// `testdata/file_name_words.txt`; palavra portuguesa lá dentro é um ato visível.
const fileNameWords = "testdata/file_name_words.txt"

// A FONTE É O `git ls-files`, e não um `WalkDir` com lista de pastas puladas.
//
// A primeira versão varria o disco e pulava `node_modules`, `dist`,
// `test-results` e mais quatro à mão. Ela passou verde no worktree — que nasce
// limpo — e reprovou **2118 segmentos** no checkout primário, onde moram
// `.claude/`, `.playwright-mcp/` e as capturas de tela de quem estava
// depurando. Nenhum desses arquivos é do repositório, e a lista à mão nunca ia
// acabar: cada ferramenta nova traz a pasta dela.
//
// Quem sabe o que é "arquivo deste repositório" é o git, então é ele que
// responde. De quebra, a lista de pastas sumiu inteira — `node_modules` e
// companhia já estão no `.gitignore`, que é onde essa informação mora uma vez
// só.
//
// A EXCEÇÃO que sobra é `engine-go/parity/`, e ela é de CONTEÚDO e não de
// ferramenta: o nome daqueles 18 arquivos é o `slug` gravado dentro do
// `_fixtures.json`, então renomeá-los é editar o dado — e o dado ali é vizinho
// do oráculo, que o `engine-go/CLAUDE.md` diz que só se regenera por ato
// deliberado. Decisão do dono, ALE-301: as 18 ficam.
//
// **Pasta de PACOTE não entra aqui**: `live/` e `platform/` seguem em
// português por decisão do glossário (§E-bis) e este guarda não as vê, porque
// ele mede o NOME DO ARQUIVO e os arquivos lá dentro já são ingleses. O
// `tabuleiro/` virou `board/` nesta mesma fatia.
const parityIsTheException = "engine-go/parity/"

// A quebra é por `.`, `_`, `-` e por camelCase, e o segmento puramente numérico
// sai fora: o `00011` de uma migração e o `2` de `sheetv2` não são palavra de
// língua nenhuma.
var (
	fileNameSeparators = regexp.MustCompile(`[._\-]+`)
	fileNameCamel      = regexp.MustCompile(`([a-z])([A-Z])`)
	fileNameDigits     = regexp.MustCompile(`^[0-9]+$`)
)

func TestNoFileIsNamedInPortuguese(t *testing.T) {
	allowed := map[string]bool{}
	raw, err := os.ReadFile(fileNameWords)
	if err != nil {
		t.Fatalf("ler %s: %v", fileNameWords, err)
	}
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			allowed[l] = true
		}
	}

	root := filepath.Join("..", "..")
	// `-z` porque nome de arquivo pode ter espaço, e `--cached` porque o que
	// vale é o que está VERSIONADO: um arquivo novo ainda não adicionado não é
	// do repositório, e um que alguém apagou sem commitar ainda é.
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}

	filesRead, segmentsRead := 0, 0
	var unknown []string
	used := map[string]bool{}

	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasPrefix(relative, parityIsTheException) {
			continue
		}
		// O PONTO da frente não isenta: `.gitignore` e `.dockerignore` são nome
		// de arquivo como qualquer outro, e a primeira versão que os pulava
		// deixava duas palavras órfãs no léxico — o guarda denunciando o próprio
		// recorte, que é o que um denominador serve para fazer.
		name := strings.TrimPrefix(filepath.Base(relative), ".")
		filesRead++
		spaced := fileNameCamel.ReplaceAllString(name, "$1 $2")
		for _, seg := range fileNameSeparators.Split(spaced, -1) {
			for _, word := range strings.Fields(seg) {
				word = strings.ToLower(word)
				if word == "" || fileNameDigits.MatchString(word) {
					continue
				}
				segmentsRead++
				if allowed[word] {
					used[word] = true
					continue
				}
				unknown = append(unknown, relative+" — "+word)
			}
		}
	}

	// O DENOMINADOR: uma lista vazia e uma raiz errada se parecem no terminal.
	// Os pisos são folgados de propósito — eles denunciam a raiz trocada e o
	// `WalkDir` que parou no primeiro diretório, não uma fatia que apagou dez
	// arquivos.
	if filesRead < 500 || segmentsRead < 1000 {
		t.Fatalf("a varredura leu %d arquivos e %d segmentos — a raiz é o primeiro suspeito", filesRead, segmentsRead)
	}

	sort.Strings(unknown)
	if len(unknown) > 0 {
		t.Errorf("segmento de nome de arquivo que a lista de PERMITIDOS não conhece — %d de %d:\n  %s\n"+
			"Nome de arquivo é IDENTIFICADOR e sai em inglês (CLAUDE.md, \"Idioma\"); a descrição de "+
			"dentro do `test('…')` é que fica em português. Se a palavra é inglesa e nova, "+
			"acrescente a linha em %s — o guarda falha no que não conhece de propósito, porque a "+
			"lista de PROIBIDOS que ele substituiu subcontou 96 arquivos para 13.",
			len(unknown), segmentsRead, strings.Join(unknown, "\n  "), fileNameWords)
	}

	// A lista também não pode ENVELHECER: palavra que não nomeia mais nada vira
	// mentira sozinha, pelo mesmo motivo da linha de base do idioma.
	var orphans []string
	for word := range allowed {
		if !used[word] {
			orphans = append(orphans, word)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("a lista de permitidos cita %d palavra(s) que não nomeiam arquivo nenhum:\n  %s\n"+
			"Tire-as de %s: um arquivo que descreve o que já não existe é defeito entregue igual "+
			"a qualquer outro.", len(orphans), strings.Join(orphans, ", "), fileNameWords)
	}
	t.Logf("nomes de arquivo: %d arquivos, %d segmentos, %d palavras no léxico", filesRead, segmentsRead, len(allowed))
}
