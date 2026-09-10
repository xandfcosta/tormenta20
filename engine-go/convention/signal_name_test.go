package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// NOME DE SINAL DO DATASTAR: `snake_case`, INGLÊS, UMA GRAFIA SÓ (ALE-301).
//
// O padrão inteiro e a razão de cada metade estão no `CLAUDE.md`, seção
// "Idioma". O que este guarda cobra é a FORMA — `snake_case` —, porque é ela
// que o parser do navegador decide, e é ela que já produziu defeito.
//
// A metade do IDIOMA não tem guarda próprio e não deve ter: "está em inglês"
// não é mecanizável sem dicionário, e a lista de proibidos que tentaria isso
// subconta em silêncio — foi o que a fatia dos nomes de arquivo mediu, 96
// arquivos contados como 13. Quem cobra o idioma aqui é a CATRACA: um sinal só
// sai da linha de base quando alguém o renomeia, e renomeá-lo passa pela regra.
//
// # Por que `snake_case`, e não camelCase nem kebab
//
// Medido no navegador, não deduzido:
//
//   - **camelCase quebra.** Chave de atributo é minusculada pelo analisador de
//     HTML. `data-bind:buscaCriatura` chega como `data-bind:buscacriatura` e o
//     Datastar liga um sinal NOVO; o declarado fica intocado e o fio leva os
//     DOIS. Estava vivo no construtor de encontros, e funcionava só porque o
//     `encoding/json` do Go casa campo sem diferenciar caixa e a chave ligada
//     vinha por último — invertida a ordem, a busca deixa de filtrar em
//     silêncio.
//   - **kebab vira outra coisa.** O Datastar transforma `-[a-z]` em maiúscula
//     por padrão (o modificador `case`), então `data-bind:creature-search` liga
//     um sinal em camelCase, e a expressão teria de lê-lo nessa outra grafia.
//     São DUAS grafias para um conceito, que é exatamente a raiz do defeito
//     acima.
//   - **`_` atravessa intacto.** Não há caixa para perder e o `-` é o único
//     caractere que a transformação toca. Medido: `data-bind:creature_search`
//     chega ao parser como `data-bind:creature_search` e o fio leva uma chave
//     só.
const signalDebt = "testdata/signal_debt.txt"

var (
	signalInExpression = regexp.MustCompile(`(?:^|[^$])\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	signalInBindKey    = regexp.MustCompile(`\bdata-(?:bind|indicator|ref|computed)[a-z-]*:([a-zA-Z0-9_-]+)`)
	signalIsSnakeCase  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

func TestNoNewSignalBreaksTheNamingStandard(t *testing.T) {
	debt := map[string]bool{}
	raw, err := os.ReadFile(signalDebt)
	if err != nil {
		t.Fatalf("ler a linha de base %s: %v", signalDebt, err)
	}
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			debt[l] = true
		}
	}

	root := filepath.Join("..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go", "*.ts", "*.tsx").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}

	found := map[string]string{}
	filesRead := 0
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		// O `_templ.go` repete o `.templ`, e o `dump.go` da seed carrega um hash
		// bcrypt — `$2a$12$Ku…` — que casa com qualquer sonda de `$nome`.
		if relative == "" || strings.HasSuffix(relative, "_templ.go") ||
			relative == "engine-go/cmd/seed/dump.go" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			// COMENTÁRIO fica de fora: os treze comentários que ensinam a
			// armadilha citam a forma errada de propósito, e um guarda que os
			// proibisse tiraria do repositório a explicação que a impede.
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			for _, rx := range []*regexp.Regexp{signalInExpression, signalInBindKey} {
				for _, m := range rx.FindAllStringSubmatch(line, -1) {
					if _, já := found[m[1]]; !já {
						found[m[1]] = relative
					}
				}
			}
		}
	}

	// O DENOMINADOR: "nenhum sinal fora do padrão" e "nenhum sinal lido" são a
	// mesma cor no terminal.
	if filesRead < 300 || len(found) < 100 {
		t.Fatalf("a varredura leu %d arquivos e achou %d sinais — a raiz é o primeiro suspeito",
			filesRead, len(found))
	}

	// O QUE ESTE GUARDA PEGA, dito sem exagero: nome NOVO fora da forma, e nome
	// baselinado que sumiu. Ele NÃO cobra tradução — um `$busca` é `snake_case`
	// válido e passaria se não estivesse na linha de base. Quem força a tradução
	// é a linha de base encolher: renomear um sinal o tira de `found`, e aí o
	// ramo `sumidos` abaixo cobra que ele saia do arquivo também.
	var novos, sumidos []string
	for nome, onde := range found {
		if debt[nome] {
			continue
		}
		if !signalIsSnakeCase.MatchString(nome) {
			novos = append(novos, nome+" — "+onde)
		}
	}
	for nome := range debt {
		if _, ainda := found[nome]; !ainda {
			sumidos = append(sumidos, nome)
		}
	}
	sort.Strings(novos)
	sort.Strings(sumidos)

	if len(novos) > 0 {
		t.Errorf("sinal fora do padrão de nome — %d:\n  %s\n"+
			"Nome de sinal é `snake_case`, em inglês, com UMA grafia em todos os canais "+
			"(CLAUDE.md, \"Idioma\"). Caixa alta em chave de atributo é minusculada pelo parser e "+
			"liga um sinal NOVO — o gesto passa a \"não fazer nada\", sem erro em lugar nenhum. "+
			"A linha de base em %s registra a dívida ANTIGA e não aceita nome novo.",
			len(novos), strings.Join(novos, "\n  "), signalDebt)
	}
	if len(sumidos) > 0 {
		t.Errorf("a linha de base cita %d sinal(is) que não existem mais:\n  %s\n"+
			"Tire-os de %s: uma catraca que não encolhe deixa de ser catraca, e um arquivo que "+
			"descreve o que já não existe é defeito entregue igual a qualquer outro.",
			len(sumidos), strings.Join(sumidos, "\n  "), signalDebt)
	}
	t.Logf("sinais: %d achados em %d arquivos, %d ainda na dívida", len(found), filesRead, len(debt))
}
