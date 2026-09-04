package convention

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// arquivosAusentesDePROPOSITO são os caminhos que a prosa cita sabendo que eles
// não estão na árvore. São de duas naturezas, e as duas são legítimas:
//
//   - o que NÃO ENTRA no git por decisão (`.env.production` é do dono da mesa);
//   - a LÁPIDE de arquivo, que é a mesma prática das lápides de teste — dizer
//     por que uma coisa saiu é informação que o `git log` esconde de quem lê o
//     arquivo.
//
// A fricção é a mesma do irmão: apagar é um ato, e o ato aparece aqui.
//
// A lista é por NOME-BASE, e isso tem um preço que vale saber: declarar um nome
// aqui perdoa QUALQUER citação a ele, inclusive um endereço vivo que apodreça
// depois. É o mesmo preço que o `tombstones` do irmão já paga, e a alternativa —
// declarar por arquivo-e-linha — envelheceria a cada edição acima da linha.
var arquivosAusentesDePROPOSITO = map[string]bool{
	".env.production": true,

	// APAGADO VAZIO na ALE-278. Quando o `book` levou as 330 linhas deste
	// arquivo (`4260797d`), sobrou nele `package api` e um `import ()` — e ele
	// atravessou três fatias assim, porque um arquivo vazio não quebra nada e
	// não aparece em diff nenhum. O `engine-go/CLAUDE.md` o nomeia para contar
	// isso, que é justamente o caso de lápide que esta lista existe para deixar
	// explícito.
	"piloto_catalogos_do_personagem.go": true,

	// O SOCKET.IO (ALE-253). Quatro arquivos que o gateway levou embora, e os
	// quatro são citados por quem ficou explicando o que SOBROU deles: o corte
	// foi pelo receptor, e dizer isso exige nomear o que foi cortado.
	"realtime_initiative.go":  true,
	"realtime_vitals.go":      true,
	"realtime_board.go":       true,
	"realtime_gating_test.go": true,

	// A SPA e o MOTOR MVP (ALE-272, d122cda3). O `engine/` guarda comentários em
	// inglês que separam a derivação de verdade do orquestrador antigo — e a
	// separação só se explica nomeando o antigo.
	"tsgen.go":       true,
	"races.go":       true,
	"compute.go":     true,
	"skills.go":      true,
	"parity_test.go": true,

	// A DIVISÃO EM PACOTES (ALE-278). Cada extração deixou, no destino, a frase
	// que diz de onde a coisa veio e o que foi MEDIDO antes de movê-la — é a
	// procedência que justifica a fronteira, e apagá-la deixaria a fronteira sem
	// razão escrita.
	"piloto_ui.templ":   true,
	"bancada_test.go":   true,
	"character_dto.go":  true,
	"creature_block.go": true,
	"busca.go":          true,

	// O BARRAMENTO DE EVENTOS (ALE-279) e a COLHEITA DO TABULEIRO. O `aviso.go`
	// é citado por um "Aqui morava", que é a forma canônica da lápide neste
	// repositório.
	"aviso.go": true,

	// O `vista.go` é a lápide mais frágil das três aqui, e fica ANOTADA em vez de
	// consertada: os comentários de `terrain.go` falam de "as duas branches" e da
	// dívida de quando elas se encontrarem. Elas JÁ se encontraram — a migração
	// virou a main. O nome do arquivo é só a ponta; a frase inteira descreve um
	// mundo de duas branches que não existe mais, e reescrevê-la é decisão de
	// quem conhece a colheita, não deste passe mecânico.
	"vista.go": true,

	// AS ROTAS JSON SEM CONSUMIDOR (ALE-277). Os dois arquivos de comando
	// traduziam o socket da SPA para HTTP na ALE-253 — 36 manipuladores, mil
	// linhas — e ficaram sem um único chamador quando as cenas em Datastar
	// passaram a mutar o estado pela porta delas. Quem os cita é quem explica o
	// que SOBROU: as regras que eles chamavam ficaram, e o `live_publish.go`
	// carrega a publicação.
	"board_commands.go":   true,
	"session_commands.go": true,

	// Os dois arquivos de TESTE de rota que morreram com as rotas na mesma
	// issue. Cada um é citado pela lápide que diz onde cada garantia dele foi
	// parar — a de `character_play_state_test.go` lista sete, uma a uma.
	"character_play_state_test.go": true,
	"creatures_http_test.go":       true,
	"invites_test.go":              true,

	// O `sse_events.go` guardava o `emitPresence` e, antes dele, o handshake do
	// fluxo da SPA. O `aovivo/stream.go` o cita para dizer de onde o laço de
	// entrega veio — e por que o transporte NÃO veio junto.
	"sse_events.go": true,
}

// oCaminhoCitado casa o que parece um arquivo DESTE repositório: um nome com
// extensão de fonte, opcionalmente com `:linha` atrás.
//
// SÓ `.go` E `.templ`, e essa restrição é o que torna o guarda possível. Medido
// na ALE-285: com as extensões todas ele acusa 325 citações, e a esmagadora
// maioria — 181 `.ts`, 20 `.js`, 17 `.tsx` — é a prática de PROCEDÊNCIA que esta
// casa valoriza. Quando o `api/auth.go` cita o `auth-user.type.ts` do Nest, ele
// está dizendo de onde a regra veio, e está certo; o arquivo nunca vai voltar a
// existir. Um guarda com 325 exceções é um guarda que alguém apaga.
//
// Nas extensões do stack VIVO a ambiguidade some: um `.go` citado que não existe
// é endereço velho, não história — e quando é história mesmo, entra na lista de
// cima, que são vinte e não trezentas.
//
// O primeiro caractere não pode ser `_`, senão `_test.go` e `_templ.go` entram —
// e esses são PADRÕES de nome, não arquivos.
var oCaminhoCitado = regexp.MustCompile(
	`\b([A-Za-z0-9][A-Za-z0-9_./-]*\.(?:go|templ))\b`)

// NENHUMA CITAÇÃO NOMEIA UM ARQUIVO QUE NÃO EXISTE (ALE-285).
//
// O irmão `TestNoCitationNamesAMissingTest` prende o mesmo defeito para NOME DE
// TESTE, e mediu 136 pendurados. Ele não olha caminho de arquivo — e por isso
// não viu que as varreduras de idioma tinham deixado 41 endereços mortos.
//
// # A origem não é a SPA: somos nós
//
// A ALE-282 e a ALE-283 renomearam identificadores e arquivos para inglês e a
// ALE-278 moveu famílias inteiras para pacotes novos. Os arquivos mudaram de
// nome; os comentários que apontavam para eles, não. O `sessao_ciclo` era citado
// em QUATRO lugares como o endereço da regra do ciclo da sessão — "a regra mora
// no ..." — e o arquivo se chama `session_lifecycle.go` desde a ALE-283. O
// `vista`, o `terreno`, o `chao`, o `aviso`, a `busca`, a `cortina`, os
// `enderecos_antigos`: a mesma história catorze vezes.
//
// É o defeito que o `CLAUDE.md` descreve — *renomear um símbolo deixa a
// explicação falando de um nome que não existe* — cometido pelas duas issues que
// mais renomearam coisas. Nenhuma das duas podia tê-lo visto: não havia guarda.
//
// > Os nomes acima estão SEM a extensão de propósito, e a razão é a primeira
// > coisa que este guarda pegou: ele reprovou a própria docstring. Declará-los em
// > `arquivosAusentesDePROPOSITO` teria resolvido e seria errado — a lista é por
// > nome-base, então perdoar o `sessao_ciclo` aqui perdoaria também o próximo
// > comentário que voltasse a apontar para ele, que é exatamente o defeito que
// > esta issue consertou em quatro lugares. Um guarda que se cala para caber no
// > próprio texto não é um guarda.
//
// # Ele compara o NOME do arquivo, e não o caminho inteiro
//
// De propósito, e a escolha tem um preço declarado: um caminho cujo DIRETÓRIO
// mudou passa, desde que o arquivo ainda exista em algum lugar. O que se caça
// aqui é o arquivo APAGADO ou RENOMEADO, que é o que uma varredura deixa para
// trás às dezenas; cobrar o caminho inteiro acusaria toda mudança de pasta e
// viraria um guarda que alguém desliga. O irmão de cima faz a mesma aposta ao
// não exigir que o teste citado esteja no pacote certo.

func TestNoCitationNamesAMissingFile(t *testing.T) {
	existentes := osNomesDeArquivoDoRepositorio(t)
	arquivos := arquivosParaCitacao(t)

	medidas := 0
	for _, caminho := range arquivos {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for numero, linha := range strings.Split(string(conteudo), "\n") {
			for _, citado := range oCaminhoCitado.FindAllString(linha, -1) {
				nome := filepath.Base(citado)
				medidas++
				if existentes[nome] || arquivosAusentesDePROPOSITO[nome] {
					continue
				}
				t.Errorf("%s:%d cita %s, que não existe na árvore.\n"+
					"Se o arquivo foi MOVIDO ou RENOMEADO, a citação acompanha; se ele foi\n"+
					"APAGADO de propósito, declare o nome em `arquivosAusentesDePROPOSITO` —\n"+
					"dizer por que uma coisa saiu é bom, e o que falta é o ato ser explícito.",
					caminho, numero+1, citado)
			}
		}
	}

	// O DENOMINADOR, pela mesma razão do irmão: uma lista de reprovados vazia e
	// um regex que parou de casar são a mesma linha verde. O piso fica longe do
	// número real de propósito — o que ele pega é a varredura QUEBRAR, não a
	// prosa encolher.
	if medidas < 300 {
		t.Fatalf("só %d citações de arquivo lidas — o guarda ficou cego", medidas)
	}
}

// osNomesDeArquivoDoRepositorio é o conjunto dos nomes-base que existem na
// árvore. `node_modules` e `.git` ficam de fora: o primeiro traria dezenas de
// milhares de nomes e faria QUALQUER citação passar, que é o modo de falha
// silenciosa deste guarda.
func osNomesDeArquivoDoRepositorio(t *testing.T) map[string]bool {
	t.Helper()
	fora := map[string]bool{}
	if err := filepath.WalkDir("../..", func(caminho string, entrada os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			if entrada.Name() == "node_modules" || entrada.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		fora[entrada.Name()] = true
		return nil
	}); err != nil {
		t.Fatalf("varrer a árvore: %v", err)
	}
	// CONTROLE: um arquivo que sabidamente existe. Sem ele, uma varredura que
	// voltasse vazia faria todas as citações reprovarem de uma vez — e o
	// diagnóstico apontaria para a prosa, que é o lugar errado.
	if !fora["GLOSSARIO.md"] {
		t.Fatal("a varredura da árvore não achou o GLOSSARIO.md: ela está medindo o lugar errado")
	}
	return fora
}
