package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TODO ID REFERENCIADO EXISTE EM ALGUM `.templ` (ALE-302).
//
// Um id apontado e ausente falha de TRÊS jeitos, e dois deles não fazem barulho
// nenhum:
//
//   - `getElementById` devolve `null` e a expressão do Datastar não faz nada —
//     o gesto morre em silêncio, que é a família inteira desta issue;
//   - `aria-labelledby` aponta para o vazio e o leitor de tela perde o nome do
//     diálogo. **Nada na tela muda**, e nenhum teste de Go vê;
//   - `for=` órfão deixa o clique no rótulo de focar o campo. Some a
//     afordância, o campo continua lá.
//
// O primeiro quebra alto o bastante para o Playwright pegar. Os outros dois não
// quebram nada, e é por eles que este guarda existe.
//
// # O que ele NÃO vê, dito de propósito
//
// O par DINÂMICO — um `id={ sceneId }` no `.templ` e um `getElementById(%q)`
// alimentado pela mesma constante — é invisível dos dois lados. Isso não é
// buraco: os dois saem da MESMA constante, então renomeá-la move as duas pontas
// de uma vez. O que o guarda pega é justamente a assimetria, que é onde o
// descuido mora: um lado literal e o outro não.
//
// Ele também NÃO cobra o contrário ("todo id declarado é referenciado"), e isso
// é escolha: um id existe legitimamente só para o CSS ou como âncora, e um
// guarda que os proibisse mandaria apagar desenho válido.
var (
	idDeclaration = regexp.MustCompile(`(?:^|\s)id="([a-zA-Z][\w-]*)"`)
	idReferences  = []*regexp.Regexp{
		regexp.MustCompile(`getElementById\(['"]([a-zA-Z][\w-]*)['"]\)`),
		regexp.MustCompile(`(?:aria-labelledby|aria-describedby|popovertarget|for)="([a-zA-Z][\w-]*)"`),
		regexp.MustCompile(`querySelector\(['"]#([a-zA-Z][\w-]*)['"]\)`),
	}
)

func TestEveryReferencedElementIdExists(t *testing.T) {
	root := filepath.Join("..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go", "*.ts").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	declared := map[string]bool{}
	referenced := map[string][]string{}
	filesRead := 0
	refsRead := 0
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		for n, linha := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(linha), "//") {
				continue
			}
			for _, m := range idDeclaration.FindAllStringSubmatch(linha, -1) {
				declared[m[1]] = true
			}
			for _, rx := range idReferences {
				for _, m := range rx.FindAllStringSubmatch(linha, -1) {
					refsRead++
					// TODOS os sítios, e não só o primeiro: quem renomeou um id
					// precisa da lista para fechar as pontas de uma vez.
					referenced[m[1]] = append(referenced[m[1]], relative+":"+strconv.Itoa(n+1))
				}
			}
		}
	}

	// O DENOMINADOR: "nenhuma referência solta" e "nenhuma referência lida" são
	// a mesma cor no terminal, e este guarda depende de seis regexes estreitos —
	// é justamente o tipo que passa verde por deixar de casar.
	// Os pisos são folgados e vêm da medição de hoje — 112 declarados e 113
	// referências. A primeira versão pediu 150 porque eu contei as ocorrências
	// COM os `_templ.go`, que o guarda exclui: o piso reprovou a árvore sã, que
	// é o jeito barato de descobrir que o denominador estava errado.
	if filesRead < 300 || len(declared) < 80 || refsRead < 90 {
		t.Fatalf("a varredura leu %d arquivos, %d ids declarados e %d referências — a raiz é o primeiro suspeito",
			filesRead, len(declared), refsRead)
	}

	var soltas []string
	for id, onde := range referenced {
		if !declared[id] {
			sort.Strings(onde)
			soltas = append(soltas, id+" — apontado em "+strings.Join(onde, ", "))
		}
	}
	sort.Strings(soltas)
	if len(soltas) > 0 {
		t.Errorf("id apontado que NÃO existe em `.templ` nenhum — %d de %d referências distintas:\n  %s\n"+
			"Um `getElementById` assim devolve `null` e o gesto não faz nada; um `aria-labelledby` assim "+
			"tira o nome do diálogo do leitor de tela, e a TELA NÃO MUDA. Se o id foi renomeado, as duas "+
			"pontas andam juntas.",
			len(soltas), len(referenced), strings.Join(soltas, "\n  "))
	}
	t.Logf("ids: %d declarados, %d referências distintas (%d no total), de %d arquivos",
		len(declared), len(referenced), refsRead, filesRead)
}
