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

// NENHUMA ROTA DO TABULEIRO CARREGA COORDENADA NO CAMINHO (ALE-307).
//
// A coordenada de um gesto viaja no CORPO (ver o `engine-go/CLAUDE.md`). O que
// este guarda impede não é a decisão — é o ESQUECIMENTO dela, e ele existe
// porque o esquecimento já aconteceu.
//
// A ALE-305 converteu a família e eu declarei "23 de 23". Era 24: a
// `/tabuleiro/colar/{x}/{y}` escapou porque é registrada com o caminho INTEIRO
// e a minha varredura procurava `base+"…"`. Um `grep` ancorado no jeito comum
// de escrever não mede o jeito incomum, e o que sobra parece resposta.
//
// Aqui o padrão casa o PARÂMETRO e não a forma de registrar: `{x}` ou `{y}` em
// qualquer rota, escrita como for.
//
// E O `{dx}` ENTROU DEPOIS, pela mesma porta. Escrito só com os nomes do ponto
// absoluto, este guarda passou VERDE por cima de `/grupo/mover/{dx}/{dy}` — o
// arrasto do grupo marcado, que monta o endereço concatenando o delta na
// expressão (`'…/grupo/mover/' + dx + '/' + dy`). É o mesmo defeito que o
// próprio cabeçalho acima descreve, cometido na linha que o descreve: eu enumerei
// as grafias que tinha na frente e a família tinha mais uma. DESLOCAMENTO é
// coordenada — o que faz dela corpo é vir do PONTEIRO, e não ser absoluta.
var coordinateInPath = regexp.MustCompile(`\{(x2?|y2?|mx|my|dx|dy)\}`)

func TestNoBoardRouteCarriesACoordinateInThePath(t *testing.T) {
	root := filepath.Join("..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	rotas, filesRead := 0, 0
	var comCoordenada []string
	registro := regexp.MustCompile(`r\.(?:Get|Post|Put|Delete|Patch)\(`)
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") ||
			strings.HasSuffix(relative, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		for n, linha := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(linha), "//") || !registro.MatchString(linha) {
				continue
			}
			rotas++
			if m := coordinateInPath.FindString(linha); m != "" {
				comCoordenada = append(comCoordenada,
					relative+":"+strconv.Itoa(n+1)+" — o parâmetro "+m)
			}
		}
	}

	// O DENOMINADOR: 276 arquivos e 193 registros hoje — os `_test.go` ficam de
	// fora, e por isso o piso de arquivos é menor que o dos outros guardas. Ele
	// denuncia o regex de registro que parou de casar, que é como este guarda
	// ficaria inerte.
	if filesRead < 200 || rotas < 120 {
		t.Fatalf("a varredura leu %d arquivos e %d registros de rota — a raiz é o primeiro suspeito",
			filesRead, rotas)
	}

	sort.Strings(comCoordenada)
	if len(comCoordenada) > 0 {
		t.Errorf("rota com coordenada no CAMINHO — %d de %d rotas:\n  %s\n"+
			"A coordenada de um gesto viaja no corpo, pelo `payload` do `@post` (ver "+
			"`engine-go/CLAUDE.md`). Se o handler também lê sinais, o payload os lista ao lado "+
			"dela — foi o que a peça avulsa e o colar precisaram.",
			len(comCoordenada), rotas, strings.Join(comCoordenada, "\n  "))
	}
	t.Logf("rotas: %d, nenhuma com coordenada no caminho, de %d arquivos", rotas, filesRead)
}
