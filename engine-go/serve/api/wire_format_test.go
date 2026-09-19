package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A GRAFIA DO FIO É MINÚSCULA, e isto é contrato com o cliente.
//
// O defeito é uma renomeação de identificador que varre a STRING da tag junto:
// `role` vira `Role` dentro de `json:"role"`, o servidor passa a mandar `Role`,
// o cliente continua lendo `role`, e o campo vira SEMPRE o zero. Nada quebra o
// build.
//
// Nenhuma barreira óbvia pega isso: teste Go atualizado JUNTO com a mudança
// passa a ler "Role" e não pode acusá-la, e o e2e afirma efeitos visíveis sem
// nunca dizer "o mestre vê os controles do mestre". Este guarda é o barato que
// pega a FAMÍLIA inteira, hoje e nas tags futuras.
func TestTheWireSpellingIsLowercase(t *testing.T) {
	t.Run("tags JSON", agrafiaDasTags)
	t.Run("caminhos de rota", agrafiaDasRotas)
}

// agrafiaDasRotas — a MESMA varredura pega os CAMINHOS, que o guarda das tags
// sozinho não vê.
//
// O chi casa caminho com sensibilidade a caixa, então `/Reset-password` é um
// 404 — e é o caminho de quem perdeu o acesso, que ninguém exercita até
// precisar.
//
// Repare no que a varredura de renomeação ACERTA: o que se parece com
// identificador dentro de string. Na mesma linha, `/Places/{placeId}/scene`
// vira maiúsculo em `Places` e fica intacto em `scene` e em `{placeId}`.
func agrafiaDasRotas(t *testing.T) {
	rota := regexp.MustCompile(`r\.(?:Get|Post|Put|Delete|Patch|Head|Options|Route|Handle|HandleFunc)\("(/[^"]*)"`)
	var sitios int
	// CAMINHA A ÁRVORE em vez de enumerar pacotes: as rotas mudam de casa, e uma
	// lista escrita à mão passa a varrer diretório vazio em silêncio. É o piso
	// abaixo que denuncia — enumerar é remendo, a caminhada é a amostragem.
	{
		var arquivos []string
		raizDoModulo, err := os.Getwd()
		if err != nil {
			t.Fatalf("achar a raiz: %v", err)
		}
		if err := filepath.WalkDir(filepath.Dir(raizDoModulo), func(caminho string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") {
				return err
			}
			arquivos = append(arquivos, caminho)
			return nil
		}); err != nil {
			t.Fatalf("caminhar a árvore: %v", err)
		}
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
	// Caminha a árvore em vez de enumerar pacotes, pela mesma razão do guarda
	// acima: enumerar é remendo, e o pacote que muda de casa sai da medição em
	// silêncio.
	raizes := []string{"."}
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
				// fonte crua acusaria a própria explicação.
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
	//
	// O piso é calibrado contra o ZERO, e não como censo: o que ele pega é um
	// regex que parou de casar, e esse caso dá zero.
	//
	// Ele DESCEU de 60 para 25, e a direção é o ponto. A conta encolhe de
	// propósito a cada fatia da migração — o DTO de fio morre junto com o
	// manipulador que o servia, e só a ALE-349 levou quatro. Um piso calibrado
	// como "metade do que existe hoje" reprovaria a próxima fatia por ela ter
	// dado certo, e a correção seria baixar o número de novo. São 55 tags agora.
	if sitios < 25 {
		t.Fatalf("só %d tags JSON em %d arquivos — o padrão parou de casar e o verde "+
			"não significa nada", sitios, visitados)
	}
	t.Logf("%d tags JSON conferidas em %d arquivos", sitios, visitados)
}
