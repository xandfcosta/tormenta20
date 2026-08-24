package api

import (
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
// COBRE OS QUATRO PACOTES, e isso foi correção da sessão da migração Datastar:
// a primeira versão varria só o `api/`, e `aovivo/` e `tabuleiro/` sozinhos têm
// 75 tags — o estado inteiro da sessão e o tabuleiro inteiro, que são fio puro.
// Um guarda que cobre o pacote onde o defeito apareceu, e não a família onde
// ele pode aparecer, é enumeração disfarçada de amostragem.
var pacotesDeFio = []string{".", "../aovivo", "../tabuleiro", "../plataforma"}

func TestTagDeFioComecaEmMinuscula(t *testing.T) {
	tag := regexp.MustCompile(`json:"([^",]+)`)

	visitados, achadas := 0, 0
	for _, pacote := range pacotesDeFio {
		arquivos, err := filepath.Glob(filepath.Join(pacote, "*.go"))
		if err != nil {
			t.Fatalf("listar %s: %v", pacote, err)
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
