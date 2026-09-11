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

// A CHAVE DO PAYLOAD TEM O NOME DO SINAL QUE ELA LÊ (ALE-306).
//
// O `payload` do `@post` SUBSTITUI os sinais em vez de acrescentá-los. Um gesto
// que precisa de um valor calculado no clique E do formulário da página tem de
// LISTAR o formulário à mão:
//
//	{payload: {from: {X: cx, Y: cy},
//	           new_token_name: $new_token_name, …}}
//
// Isso é uma grafia A MAIS de cada nome de sinal, num lugar que o `grep` de
// `$nome` não distingue de um leitor qualquer. E o modo de falhar é o da
// família: `new_token_name: $new_token_look` manda o valor errado no campo
// certo, o servidor aceita, e a peça nasce com a aparência no nome — sem erro em
// lugar nenhum.
//
// # O que ele cobra, e por que a primeira versão era larga demais
//
// Ele cobra a igualdade SÓ quando a chave é ela mesma um nome de sinal. Escrito
// como "toda chave tem de casar com o sinal que lê", o guarda reprovou três
// pares CERTOS na primeira execução: `kind: $tool`, `shape: $template` — ali o
// campo do corpo e o sinal são conceitos diferentes de propósito (a ferramenta
// que o dedo segura contra a espécie que o corpo carrega).
//
// A regra estreita pega o que interessa e só isso: se a chave `new_token_name`
// existe COMO SINAL, então ela tem de carregar `$new_token_name` e não
// `$new_token_look`. É o copiar-colar dentro de uma família de sinais irmãos —
// que é exatamente onde o olho escorrega.
//
// Ele NÃO cobra que o payload esteja completo: ligar o sítio do payload ao
// handler que o lê exige seguir a rota, e uma varredura que chutasse isso
// acusaria o certo.
var (
	payloadStart = regexp.MustCompile(`payload:`)
	payloadPair  = regexp.MustCompile(`([a-zA-Z_][\w]*)\s*:\s*\$([a-zA-Z_][\w]*)\b`)
)

func TestEveryPayloadKeyMatchesTheSignalItReads(t *testing.T) {
	root := filepath.Join("..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	// OS SINAIS DA ÁRVORE, colhidos como o `TestNoNewSignalBreaksTheNamingStandard`
	// os colhe: é essa lista que separa "chave que também é sinal" de "campo de
	// corpo com nome próprio".
	sinais := map[string]bool{}
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		for _, m := range signalInExpression.FindAllStringSubmatch(string(body), -1) {
			sinais[m[1]] = true
		}
	}

	pares, filesRead := 0, 0
	var tortos []string
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		// COMENTÁRIO fora, e a primeira versão sem isso acusou o próprio
		// cabeçalho deste arquivo: o exemplo errado está escrito ali de
		// propósito, para quem lê entender o que se evita. É a mesma fresta que
		// os treze comentários da armadilha do Datastar pedem.
		texto := strings.Join(semComentario(strings.Split(string(body), "\n")), "\n")
		// A VARREDURA É SOBRE O TEXTO INTEIRO e não linha a linha, e isso é
		// conserto de um cego: a expressão do payload é uma CONCATENAÇÃO de Go,
		// e o `payload:` mora na primeira linha enquanto os pares moram na
		// segunda e na terceira. Ancorado por linha, o guarda leu ZERO pares e
		// só não passou verde porque o piso do denominador estava lá.
		for _, corte := range payloadStart.FindAllStringIndex(texto, -1) {
			janela := texto[corte[0]:min(len(texto), corte[0]+400)]
			for _, m := range payloadPair.FindAllStringSubmatch(janela, -1) {
				if !sinais[m[1]] {
					// A chave é campo de corpo com nome próprio (`kind`,
					// `shape`), e não há o que casar.
					continue
				}
				pares++
				if m[1] != m[2] {
					linha := strings.Count(texto[:corte[0]], "\n") + 1
					tortos = append(tortos, relative+":"+strconv.Itoa(linha)+
						" — a chave `"+m[1]+"` carrega o sinal `$"+m[2]+"`")
				}
			}
		}
	}

	// O DENOMINADOR: hoje são três pares, todos num sítio só. O piso é 1 porque
	// o que ele denuncia é o regex que parou de casar, não uma fatia que tirou
	// um campo — e um piso alto aqui envelheceria no primeiro payload novo.
	if filesRead < 300 || pares < 1 {
		t.Fatalf("a varredura leu %d arquivos e %d pares `chave: $sinal` em payload — a raiz é o primeiro suspeito",
			filesRead, pares)
	}

	sort.Strings(tortos)
	if len(tortos) > 0 {
		t.Errorf("chave de payload que não tem o nome do sinal que lê — %d de %d:\n  %s\n"+
			"O servidor aceita o valor errado no campo certo e nada estoura. Se a chave e o sinal "+
			"são MESMO diferentes, o lugar de dizer isso é um campo de corpo com nome próprio, "+
			"não um par que parece igual e não é.",
			len(tortos), pares, strings.Join(tortos, "\n  "))
	}
	t.Logf("pares `chave: $sinal` em payload: %d, todos casados, de %d arquivos", pares, filesRead)
}
