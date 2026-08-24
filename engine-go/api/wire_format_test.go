package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A GRAFIA DO FIO É MINÚSCULA, e isto é contrato com o cliente (ALE-263).
//
// A ALE-254 renomeou identificadores ao partir o `api/` em quatro pacotes, e a
// renomeação varreu a STRING da tag junto: `role` virou `Role` dentro de
// `json:"role"` em sete lugares. O efeito no produto foi que o
// `GET /campaigns/{id}` passou a mandar `Role`, a SPA continuou lendo `role`, e
// o `isGm()` do rastreador virou SEMPRE FALSO — o mestre recebia a visão de
// jogador na mesa ao vivo. Um dos sete era corpo de ENTRADA, então trocar papel
// de membro também parou.
//
// Três barreiras existiam e nenhuma pegou:
//
//   - os testes Go foram atualizados JUNTO, e passaram a ler "Role": teste que
//     muda com a mudança não pode acusá-la;
//   - o gerador de tipos da fronteira cobre só o que o WASM troca, não os DTOs
//     HTTP;
//   - o e2e afirma efeitos visíveis nas DUAS visões — condição, fila,
//     tabuleiro —, e nenhuma asserção diz "o mestre vê os controles do mestre".
//
// Este guarda é o barato que pega a FAMÍLIA: um e2e da visão do mestre custaria
// minutos e cobriria um caso; isto custa milissegundos e cobre toda tag futura.
func TestAGrafiaDoFioEMinuscula(t *testing.T) {
	raizes := []string{".", "../aovivo", "../tabuleiro", "../plataforma"}
	// A tag pode vir com opções (`json:"nome,omitempty"`); o que importa é a
	// primeira letra do NOME. `json:"-"` é descarte e não é nome.
	tag := regexp.MustCompile("`[^`]*json:\"([A-Za-z][^\",]*)")

	var visitados, sitios int
	for _, raiz := range raizes {
		arquivos, err := filepath.Glob(filepath.Join(raiz, "*.go"))
		if err != nil {
			t.Fatalf("listar %s: %v", raiz, err)
		}
		for _, caminho := range arquivos {
			conteudo, err := os.ReadFile(caminho)
			if err != nil {
				t.Fatalf("ler %s: %v", caminho, err)
			}
			visitados++
			for i, linha := range strings.Split(string(conteudo), "\n") {
				// Comentário fora antes de medir: o cabeçalho acima cita
				// `json:"role"` para explicar o defeito, e um guarda que lê a
				// fonte crua acusaria a própria explicação. A sessão irmã
				// entregou um guarda irmão VERMELHO sobre o próprio texto hoje,
				// e só a prova de vermelho revelou.
				if j := strings.Index(linha, "//"); j >= 0 {
					linha = linha[:j]
				}
				for _, m := range tag.FindAllStringSubmatch(linha, -1) {
					sitios++
					nome := m[1]
					if nome[0] >= 'A' && nome[0] <= 'Z' {
						t.Errorf("%s:%d — a tag `json:%q` começa com maiúscula. O fio é "+
							"minúsculo e o cliente lê assim; uma renomeação de identificador "+
							"que varra a string da tag quebra o contrato sem quebrar o build.",
							caminho, i+1, nome)
					}
				}
			}
		}
	}
	// CONTROLE: sem ele, um regex que parou de casar diria verde sobre nada.
	if sitios < 100 {
		t.Fatalf("só %d tags JSON em %d arquivos — o padrão parou de casar e o verde "+
			"não significa nada", sitios, visitados)
	}
	t.Logf("%d tags JSON conferidas em %d arquivos", sitios, visitados)
}
