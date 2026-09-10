package convention

import (
	"io/fs"
	"os"
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

// dirsOutsideTheSweep são as que não têm nome de arquivo NOSSO dentro.
//
// `parity/` é a única EXCEÇÃO DE CONTEÚDO, e a razão é que o nome não é um
// identificador solto: ele é o `slug` gravado dentro do `_fixtures.json`, então
// renomear o arquivo é editar o dado — e o dado ali é vizinho do oráculo, que o
// `engine-go/CLAUDE.md` diz que só se regenera por ato deliberado. Decisão do
// dono, ALE-301: as 18 ficam.
//
// As outras são produto de build, dependência ou banco. **Pasta de PACOTE não
// entra aqui**: `aovivo/` e `plataforma/` seguem em português por decisão do
// glossário (§E-bis) e este guarda não as vê, porque ele mede o NOME DO ARQUIVO
// e os arquivos lá dentro já são ingleses. O `tabuleiro/` virou `board/` nesta
// mesma fatia.
var dirsOutsideTheSweep = map[string]bool{
	".git": true, "node_modules": true, "dist": true, "test-results": true,
	"playwright-report": true, "backups": true, ".auth": true,
}

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
	parity := filepath.Join(root, "engine-go", "parity")
	filesRead, segmentsRead := 0, 0
	var unknown []string
	used := map[string]bool{}

	err = filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// O nome é conferido ANTES do `IsDir`, e não é preciosismo: num
		// worktree do git o `.git` é um ARQUIVO apontando para o gitdir, não uma
		// pasta. A versão que só olhava diretórios passava no CI (clone normal,
		// `.git` é pasta) e reprovava na bancada — o pior par possível, porque o
		// verde de lá é o que autoriza o merge.
		if dirsOutsideTheSweep[item.Name()] {
			if item.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if item.IsDir() {
			if path == parity {
				return fs.SkipDir
			}
			return nil
		}
		// O banco do e2e e os despejos vivem em `engine-go/data`; o catálogo
		// vive em `engine-go/catalog/data` e É medido. Por isso o corte é por
		// CAMINHO e não por nome de pasta — a versão por nome apagaria da
		// medição justamente os seis arquivos que esta fatia renomeou.
		if strings.HasPrefix(path, filepath.Join(root, "engine-go", "data")) {
			return nil
		}
		// O PONTO da frente não isenta: `.gitignore` e `.dockerignore` são nome
		// de arquivo como qualquer outro, e a primeira versão que os pulava
		// deixava duas palavras órfãs no léxico — o guarda denunciando o próprio
		// recorte, que é o que um denominador serve para fazer.
		name := strings.TrimPrefix(item.Name(), ".")
		filesRead++
		relative := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, root)), "/")
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
		return nil
	})
	if err != nil {
		t.Fatalf("varrer %s: %v", root, err)
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
