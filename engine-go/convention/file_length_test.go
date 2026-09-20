package convention

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// theCeiling é o limite do guia, e ele não é sobre leitura.
//
// O teto de linhas por FUNÇÃO saiu do `CLAUDE.md` de propósito — ele picava uma
// tarefa que é uma coisa só em dez funções com nome inventado. O de ARQUIVO
// ficou, e com outra razão escrita: *"arquivo é unidade de RESPONSABILIDADE e
// de conflito de merge, não de leitura"*.
const theCeiling = 500

// NENHUM ARQUIVO NOVO PASSA DE 500 LINHAS, E A DÍVIDA VELHA SÓ ENCOLHE.
//
// # Por que catraca e não faxina
//
// A regra estava escrita e valia para os arquivos que alguém apontou: dezessete
// passavam do teto quando este guarda nasceu, um deles com QUATRO vezes o
// limite. Uma convenção sem varredura é aplicada por lembrança, e lembrança não
// é mecanismo (ALE-360).
//
// Consertar dezessete numa fatia seria uma faxina grande e arriscada; não
// consertar nenhum seria dívida registrada, que o guia chama de "parecer
// resolvido". A catraca faz as duas: trava o crescimento hoje e deixa a dívida
// visível, encolhendo quando alguém passar por perto.
//
// # As duas direções, e a segunda é a que importa
//
// Arquivo NOVO acima do teto reprova com o nome dele. E arquivo da base que
// ENCOLHEU abaixo do teto também reprova — pedindo que ele saia da lista. Sem
// essa metade, a base vira um arquivo que afirma uma dívida que já foi paga, e
// ninguém a relê.
func TestNoNewFileGoesOverTheLineCeiling(t *testing.T) {
	raiz, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	base := longFileBaseline(t)
	medidos := 0
	acimaDoTeto := map[string]bool{}

	err = filepath.WalkDir(raiz, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isHandWritten(caminho) {
			return err
		}
		rel, _ := filepath.Rel(raiz, caminho)
		rel = filepath.ToSlash(rel)
		medidos++
		linhas, err := countLines(caminho)
		if err != nil {
			return err
		}
		if linhas <= theCeiling {
			return nil
		}
		acimaDoTeto[rel] = true
		if base[rel] {
			return nil
		}
		t.Errorf("%s tem %d linhas, e o teto é %d.\n"+
			"Arquivo é unidade de RESPONSABILIDADE: se ele passou do teto, ele ganhou uma\n"+
			"segunda razão para mudar. Divida por essa razão — e se a divisão não tiver\n"+
			"nome, ela não é divisão.", rel, linhas, theCeiling)
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// A DÍVIDA QUE FOI PAGA sai da lista, e o guarda cobra isso: uma base que só
	// cresce é uma base que ninguém relê.
	pagos := []string{}
	for arquivo := range base {
		if !acimaDoTeto[arquivo] {
			pagos = append(pagos, arquivo)
		}
	}
	sort.Strings(pagos)
	for _, arquivo := range pagos {
		t.Errorf("%s está na linha de base de arquivos longos e hoje cabe no teto.\n"+
			"Tire a linha dele de `convention/testdata/long_files.txt` — a base registra\n"+
			"dívida ANTIGA, e uma que afirma o que já foi pago vira mentira sozinha.", arquivo)
	}

	// O DENOMINADOR: uma varredura que não abriu arquivo e uma árvore sem
	// violação se parecem no terminal.
	if medidos < 300 {
		t.Fatalf("o guarda leu só %d arquivos — está medindo a árvore errada", medidos)
	}
	t.Logf("teto de %d linhas: %d arquivos medidos, %d na dívida", theCeiling, medidos, len(base))
}

// # Os `.templ` entram, e a divisão deles é POSSÍVEL
//
// Seis `.templ` estão na dívida, com o `board.templ` em 2084 linhas — quatro
// vezes o teto. Achei que o `templ` não tivesse como dividi-los por não ter
// `include`; está errado, e conferi: ele resolve componente por PACOTE, não por
// arquivo. O `boardTable` mora no `board.templ` e é chamado do `table.templ` e
// do `draft.templ`, hoje. Mover componentes para um arquivo irmão do mesmo
// pacote não pede nada — o que pede é a razão do corte (ALE-360).
//
// isHandWritten separa o que é ESCRITO à mão do que é gerado.
//
// Gerado fica de fora porque o teto é sobre responsabilidade, e ninguém escolhe
// a responsabilidade de um arquivo que uma ferramenta emite: o `_templ.go` é o
// `.templ` compilado, e o `sqlcgen` é o `query.sql`. Cobrar deles seria cobrar
// da ferramenta.
func isHandWritten(caminho string) bool {
	if strings.HasSuffix(caminho, "_templ.go") || strings.Contains(caminho, "/sqlcgen/") {
		return false
	}
	// TESTE fica de fora, e é decisão e não concessão: um caso vem com o
	// cabeçalho que explica o mecanismo, e o guia PEDE esse cabeçalho. Cobrar o
	// teto deles empurraria na direção contrária à seção "Comentários".
	if strings.HasSuffix(caminho, "_test.go") {
		return false
	}
	return strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, ".templ")
}

func countLines(caminho string) (int, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n := 0
	leitor := bufio.NewScanner(f)
	leitor.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for leitor.Scan() {
		n++
	}
	return n, leitor.Err()
}

// longFileBaseline lê a dívida registrada — um caminho por linha.
func longFileBaseline(t *testing.T) map[string]bool {
	t.Helper()
	bruto, err := os.ReadFile(filepath.Join("testdata", "long_files.txt"))
	if err != nil {
		t.Fatalf("ler a linha de base: %v", err)
	}
	base := map[string]bool{}
	for _, linha := range strings.Split(string(bruto), "\n") {
		if linha = strings.TrimSpace(linha); linha != "" {
			base[linha] = true
		}
	}
	return base
}
