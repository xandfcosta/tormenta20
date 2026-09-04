package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A TAG JSON É CONTRATO COM O CLIENTE, E ELA COMEÇA EM MINÚSCULA.
//
// Este guarda nasceu de um defeito que chegou à `main` e que nenhuma das
// barreiras existentes viu (ALE-254). O laço que exportava símbolos durante a
// extração dos contextos renomeou `role` para `Role` — e a expressão casou
// DENTRO da string da tag, virando a grafia capitalizada em sete lugares.
//
// O efeito era grave e silencioso: a SPA lê `campanha.role` em 22 pontos, e com
// a tag capitalizada o campo chega `undefined`. O `isGm()` passava a ser SEMPRE
// falso, então o MESTRE recebia a visão de jogador na mesa ao vivo. Um deles
// (`members.go`) era corpo de ENTRADA, então trocar o papel de um membro
// simplesmente parava de funcionar.
//
// POR QUE NADA PEGOU, e é isto que justifica um guarda novo em vez de confiar
// nos que existem:
//
//   - Os testes Go foram renomeados PELO MESMO laço. O `authz_http_test.go`
//     afirmava o campo em minúscula e passou a afirmá-lo capitalizado — ele
//     teria pegado o defeito e foi cegado junto com ele. Teste que muda com a
//     mudança não acusa a mudança.
//   - O gerador de tipos da fronteira não alcança: o `engine-types.ts` cobre o
//     que o WASM devolve e recebe, e os DTOs HTTP ficam de fora.
//   - Entre os dois não havia nada.
//
// O guarda é uma leitura de fonte e pega a FAMÍLIA inteira: o próximo rename
// que varrer uma tag junto morre aqui, com o nome do arquivo e do campo. Não há
// lista de campos a manter — amostragem e não enumeração.
//
// A regra é do glossário: **a fronteira fala inglês** — nome de tabela, campo
// JSON, evento SSE e rota HTTP — e o cliente foi escrito contra a grafia
// minúscula. Trocar a caixa quebra cliente e migração, e o ganho é zero.
//
// ELE CAMINHA A ÁRVORE, e não uma lista de pacotes (ALE-278).
//
// A versão anterior enumerava quatro — `api`, `aovivo`, `tabuleiro`,
// `plataforma` —, e isso já era a correção de uma que varria só o `api`. A
// enumeração quebrou do jeito previsto: quando a Mesa virou `web/table` as tags
// dela saíram da lista, a contagem caiu abaixo do piso e o guarda falhou ALTO.
// Foi sorte de o piso existir; sem ele, o guarda teria seguido verde medindo
// menos.
//
// **Enumerar é remendo, e o que restaura a amostragem é a caminhada.** A tag
// JSON é contrato com o cliente em qualquer pacote que a escreva, e o pacote
// novo nasce medido.

// A ROTA TAMBÉM É FIO, E TAMBÉM COMEÇA EM MINÚSCULA.
//
// O mesmo laço da ALE-254 que capitalizou as tags capitalizou OITO ROTAS, e
// isto só apareceu depois — porque no commit daquele trabalho eu rodei a suíte
// Go e não a do front, e quem acusa rota errada é o `realtime-wire.test.ts`,
// que compara o roteador do chi com a tabela de comandos do cliente.
//
// O estrago era maior que o das tags: `/Populate`, `/Places` e companhia
// quebram sete comandos da mesa ao vivo, e o `/Reset-password` do `server.go`
// quebra REDEFINIR SENHA — uma rota que ninguém exercita no dia a dia e que só
// falha quando alguém precisa dela.
//
// Fica no mesmo arquivo que o guarda das tags de propósito: é a mesma família
// (rename que varre uma STRING junto) e a mesma resposta (conserte a string,
// não o guarda). Separá-los faria parecer que são dois problemas.
func TestAWireRouteStartsLowercase(t *testing.T) {
	rota := regexp.MustCompile(`r\.(?:Get|Post|Put|Patch|Delete|Route)\("(/[^"]*)"`)

	arquivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listar o pacote: %v", err)
	}

	visitadas := 0
	for _, nome := range arquivos {
		bruto, err := os.ReadFile(nome)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for linha, texto := range strings.Split(string(bruto), "\n") {
			if corte := strings.Index(texto, "//"); corte >= 0 {
				texto = texto[:corte]
			}
			for _, m := range rota.FindAllStringSubmatch(texto, -1) {
				caminho := m[1]
				visitadas++
				for _, pedaco := range strings.Split(strings.Trim(caminho, "/"), "/") {
					// `{id}` é parâmetro e segue o nome do campo, não a rota.
					if pedaco == "" || strings.HasPrefix(pedaco, "{") {
						continue
					}
					// Qualquer maiúscula, não só a primeira: o defeito real
					// incluía `password-Reset`, onde o segmento COMEÇA minúsculo e
					// a varredura capitalizou a palavra depois do hífen. Um guarda
					// que olhasse só a inicial passaria verde sobre ele — e passou,
					// até o `grep` manual achar a nona rota.
					if strings.ContainsAny(pedaco, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
						t.Errorf("%s:%d: a rota %q tem segmento em MAIÚSCULA (%q).\n"+
							"O cliente chama a grafia minúscula; capitalizar vira 404, e o\n"+
							"404 chega à tela como funcionalidade que sumiu.",
							nome, linha+1, caminho, pedaco)
					}
				}
			}
		}
	}

	if visitadas < 40 {
		t.Fatalf("guarda cego: só %d rotas reconhecidas — o padrão parou de casar", visitadas)
	}
}

func TestAWireTagStartsLowercase(t *testing.T) {
	tag := regexp.MustCompile(`json:"([^",]+)`)

	visitados, achadas := 0, 0
	{
		var arquivos []string
		raiz, err := os.Getwd()
		if err != nil {
			t.Fatalf("achar a raiz: %v", err)
		}
		if err := filepath.WalkDir(filepath.Dir(raiz), func(caminho string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") {
				return err
			}
			arquivos = append(arquivos, caminho)
			return nil
		}); err != nil {
			t.Fatalf("caminhar a árvore: %v", err)
		}
		for _, nome := range arquivos {
			bruto, err := os.ReadFile(nome)
			if err != nil {
				t.Fatalf("ler %s: %v", nome, err)
			}
			visitados++
			for linha, texto := range strings.Split(string(bruto), "\n") {
				// Comentário não é contrato. Sem isto o guarda lê o próprio texto
				// explicativo — que cita a tag defeituosa — e falha para sempre
				// sobre si mesmo. Achado ao provar o vermelho: ele já estava
				// vermelho ANTES da sabotagem, o que denunciou o autoexame.
				if corte := strings.Index(texto, "//"); corte >= 0 {
					texto = texto[:corte]
				}
				for _, m := range tag.FindAllStringSubmatch(texto, -1) {
					campo := m[1]
					achadas++
					if campo == "" || campo == "-" {
						continue
					}
					if campo[0] >= 'A' && campo[0] <= 'Z' {
						t.Errorf("%s:%d: a tag `json:%q` começa em MAIÚSCULA.\n"+
							"O cliente lê a grafia minúscula; capitalizar entrega o campo como\n"+
							"`undefined` sem erro nenhum. Se isto veio de um rename automático,\n"+
							"a expressão casou dentro da string da tag — conserte a tag, não o guarda.",
							nome, linha+1, campo)
					}
				}
			}
		}
	}

	// Ausência não é aprovação. O piso é medido e não redondo: os quatro
	// pacotes tinham ~600 tags quando este guarda foi escrito, então 300 é
	// metade — folga para o código encolher, e barulho na hora em que o padrão
	// parar de casar. Um guarda que varre zero arquivos passa verde sobre nada.
	if visitados == 0 || achadas < 300 {
		t.Fatalf("guarda cego: %d arquivos, %d tags — o padrão parou de casar", visitados, achadas)
	}
}
