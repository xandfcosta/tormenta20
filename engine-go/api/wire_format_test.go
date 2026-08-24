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
	t.Run("tags JSON", agrafiaDasTags)
	t.Run("caminhos de rota", agrafiaDasRotas)
}

// agrafiaDasRotas — a MESMA varredura pegou os caminhos, e o guarda das tags
// sozinho não os via.
//
// Oito sítios: `/Populate`, `/Places`, `/Reopen` e, o pior, `/Reset-password`.
// O chi casa caminho com sensibilidade a caixa, então cada um era um 404. O da
// senha é o que dói: é o caminho de quem perdeu o acesso, e ninguém o exercita
// até precisar.
//
// Sete dos oito o `realtime-wire.test.ts` acusou — ele compara o que o cliente
// chama com o que o roteador registra, e foi por isso que a ALE-253 o trocou de
// "nome de evento" para "método + caminho". O oitavo escapou porque ele só
// cobre rotas de sessão. Este aqui varre o repositório inteiro.
//
// Repare no que a varredura de renomeação ACERTA: o que se parece com
// identificador dentro de string. Na mesma linha, `/Places/{placeId}/scene`
// virou maiúsculo em `Places` e ficou intacto em `scene` e em `{placeId}`.
func agrafiaDasRotas(t *testing.T) {
	rota := regexp.MustCompile(`r\.(?:Get|Post|Put|Delete|Patch|Head|Options|Route|Handle|HandleFunc)\("(/[^"]*)"`)
	var sitios int
	for _, raiz := range []string{".", "../aovivo", "../tabuleiro", "../plataforma", "../cmd/api"} {
		arquivos, _ := filepath.Glob(filepath.Join(raiz, "*.go"))
		for _, caminho := range arquivos {
			conteudo, err := os.ReadFile(caminho)
			if err != nil {
				t.Fatalf("ler %s: %v", caminho, err)
			}
			for i, linha := range strings.Split(string(conteudo), "\n") {
				if j := strings.Index(linha, "//"); j >= 0 {
					linha = linha[:j]
				}
				for _, m := range rota.FindAllStringSubmatch(linha, -1) {
					sitios++
					for _, seg := range strings.Split(strings.Trim(m[1], "/"), "/") {
						if seg == "" || strings.HasPrefix(seg, "{") {
							continue
						}
						if seg[0] >= 'A' && seg[0] <= 'Z' {
							t.Errorf("%s:%d — o caminho %q tem o segmento %q em maiúscula. "+
								"O chi casa com sensibilidade a caixa: isto é um 404 para o "+
								"cliente, que chama minúsculo.", caminho, i+1, m[1], seg)
						}
					}
				}
			}
		}
	}
	if sitios < 40 {
		t.Fatalf("só %d rotas casadas — o padrão parou de casar e o verde não vale", sitios)
	}
	t.Logf("%d caminhos de rota conferidos", sitios)
}

func agrafiaDasTags(t *testing.T) {
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
