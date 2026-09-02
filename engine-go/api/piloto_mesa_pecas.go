package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
	"t20engine/tabuleiro"
)

// PÔR NO MAPA (ALE-264, item 5) — ver a linha do GLOSSARIO.
//
// O gesto que faltava: o tabuleiro do piloto desenhava peças desde o `33380d6`,
// mas só nascia peça por `curl`. Aqui ele ganha a afordância, e o servidor já
// tinha tudo — `BoardStore.Populate` existe, é idempotente e tem guarda próprio
// (`board_populate_test.go`).
//
// O NOME não é `trazOGrupo` porque esse já existe neste pacote e leva o grupo
// para a FILA. Mesmo verbo, destinos diferentes: é a colisão que a linha do
// glossário nasceu para prender.

// candidatoAoMapa é uma linha do diálogo — um combatente da fila e o que a tela
// precisa saber sobre ele.
type candidatoAoMapa struct {
	ID   string
	Nome string
	// Ficha responde "é ficha de jogador ou é NPC?" (`type == "character"`), que
	// é o predicado com que o SERVIDOR escolhe o lado do mapa. Usar o mesmo aqui
	// é o que faz o atalho pôr as peças exatamente na fileira do grupo — decisão
	// do dono na ALE-204, registrada na colisão C4 do GLOSSARIO.
	Ficha bool
	// NoMapa: já tem peça. A linha continua aparecendo, marcada e travada, em vez
	// de sumir: esconder faria o mestre procurar um nome que ele acabou de ver na
	// fila, e trazer de novo não faria nada de qualquer forma.
	NoMapa bool
}

// candidatosAoMapa lista a fila com quem já está no mapa marcado.
//
// A ordem é a da FILA e não alfabética: é a ordem em que o mestre acabou de ler
// os nomes na tela ao lado, e reordenar aqui faria ele procurar duas vezes.
func candidatosAoMapa(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState) []candidatoAoMapa {
	if b == nil || st == nil {
		return nil
	}
	jaTemPeca := map[string]bool{}
	for i := range b.Tokens {
		if id := b.Tokens[i].EntryID; id != nil {
			jaTemPeca[*id] = true
		}
	}
	lista := make([]candidatoAoMapa, 0, len(st.Initiative))
	for _, entrada := range st.Initiative {
		lista = append(lista, candidatoAoMapa{
			ID:     entrada.ID,
			Nome:   entrada.Label,
			Ficha:  entrada.Type == "character",
			NoMapa: jaTemPeca[entrada.ID],
		})
	}
	return lista
}

// fichasForaDoMapa são os ids que o atalho e a abertura do diálogo escolhem.
//
// Devolve uma LISTA e não um conjunto porque ela vai virar texto numa expressão
// do navegador — e a ordem estável é o que faz duas aberturas seguidas do
// diálogo desenharem a mesma coisa.
func fichasForaDoMapa(candidatos []candidatoAoMapa) []string {
	var ids []string
	for _, c := range candidatos {
		if c.Ficha && !c.NoMapa {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// poeNoMapa faz nascer as peças escolhidas.
//
// Duas mutações e não uma, e a segunda é a que se perde num porte apressado: o
// `Populate` cria as peças e o `SetSpeeds` grava o ORÇAMENTO de movimento delas.
// Sem o segundo a peça nasce no mapa sem deslocamento, o alcance não acende e o
// jogador vê uma peça que não anda — um meio-recurso que ninguém reporta porque
// parece regra. É o que o `handleBoardPopulate` da SPA já fazia.
func poeNoMapa(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	escolhidos, err := escolhidosDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	board, err := st.boards.Populate(
		c.R.Context(), c.SessionID, c.TabuleiroID, st.sessions.GetState(c.SessionID), escolhidos,
	)
	if err != nil {
		return board, err
	}
	if speeds := st.speedsForBoard(board); len(speeds) > 0 {
		// O erro do deslocamento NÃO derruba o comando: as peças já nasceram e a
		// mesa precisa vê-las. Devolver erro aqui deixaria o mestre achando que
		// nada aconteceu sobre um mapa que mudou.
		if comVelocidade, err := st.boards.SetSpeeds(c.R.Context(), c.SessionID, c.TabuleiroID, speeds); err == nil {
			board = comVelocidade
		}
	}
	return board, nil
}

// escolhidosDosSinais lê a escolha do diálogo.
//
// O sinal é UMA string com os ids separados por vírgula, e ela é segura porque
// id de combatente é UUID (`aovivo.NewUUID`) — não há vírgula dentro de um id
// para partir a lista no meio. Um sinal por candidato seria um sinal por nome na
// fila, criados e destruídos a cada remendo da cena.
//
// VAZIO É ERRO, e a distinção importa: `EntrySelection` nil significa TODAS, que
// é exatamente o padrão inseguro que a ALE-204 tirou do app. Se a leitura falha
// ou ninguém foi escolhido, o comando recusa em vez de cair no "traz todo mundo"
// — o vilão do terceiro turno não vai para o mapa por causa de um sinal perdido.
func escolhidosDosSinais(r *http.Request) (tabuleiro.EntrySelection, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais struct {
		Escolhidos string `json:"escolhidosdomapa"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return nil, fmt.Errorf("não entendi quem pôr no mapa: %v", err)
	}
	escolha := tabuleiro.EntrySelection{}
	for _, id := range strings.Split(sinais.Escolhidos, ",") {
		if id = strings.TrimSpace(id); id != "" {
			escolha[id] = true
		}
	}
	if len(escolha) == 0 {
		return nil, errors.New("escolha ao menos um combatente para pôr no mapa")
	}
	return escolha, nil
}

// ---------------------------------------------------------------------------
// As expressões que o navegador roda. Elas moram aqui e não no `.templ` para o
// desenho ficar legível — é a mesma divisão do `escolheOPincel`.

// listaDeIDs escreve os ids do jeito que o sinal os guarda.
func listaDeIDs(ids []string) string { return strings.Join(ids, ",") }

// fichasDoDialogo lê no DOM os ids das fichas que ainda não têm peça.
//
// LER O DIÁLOGO em vez de escrever os ids no botão não é preferência de estilo:
// o botão mora na região do MAPA, e id de combatente é dado da FILA. Embutido
// ali, qualquer mudança na fila mudaria o HTML do mapa e o remendo trocaria a
// peça debaixo do dedo do mestre no meio do arrasto — foi exatamente isso que o
// `TestATrackerChangeDoesNotPatchTheMap` acusou na primeira versão.
const fichasDoDialogo = "[...document.querySelectorAll('#por-no-mapa [data-ficha]')]" +
	".map((e) => e.dataset.id).join(',')"

// abreOPorNoMapa recomeça a escolha no padrão SEGURO e abre o diálogo.
//
// Cada abertura reescreve o sinal em vez de continuar de onde parou, e isso não
// é limpeza: o rascunho da vez anterior pode ter o vilão marcado, e um diálogo
// que lembra a escolha de dois minutos atrás põe a emboscada no mapa com um
// clique em "Pôr no mapa" que o mestre acha que está confirmando outra coisa.
func abreOPorNoMapa() string {
	return "$escolhidosdomapa = " + fichasDoDialogo +
		"; document.getElementById('por-no-mapa').showModal()"
}

// alternaNoMapa liga ou desliga um id na escolha.
//
// O `filter(Boolean)` é o que impede a vírgula solta: sem ele, desmarcar o único
// escolhido deixaria a string `""` virar `[""]` ao voltar, e o servidor receberia
// um id vazio.
func alternaNoMapa(id string) string {
	return fmt.Sprintf(
		"$escolhidosdomapa = ($escolhidosdomapa.split(',').filter(Boolean).includes(%q)"+
			" ? $escolhidosdomapa.split(',').filter((v) => v && v !== %q)"+
			" : [...$escolhidosdomapa.split(',').filter(Boolean), %q]).join(',')",
		id, id, id,
	)
}

// estaEscolhido é a pergunta que pinta o crachá.
func estaEscolhido(id string) string {
	return fmt.Sprintf("$escolhidosdomapa.split(',').includes(%q)", id)
}

// comandoDePorNoMapa posta a escolha.
func comandoDePorNoMapa(v tabuleiroView) string {
	return comandoDoTabuleiroDaCena(v, "pecas")
}

// atalhoDasFichas é o clique DIREITO: põe só as fichas, sem diálogo (ALE-204).
//
// O gesto nunca é o único caminho — abrir o diálogo e confirmar faz exatamente
// isto, porque a abertura marca as mesmas fichas. O `preventDefault` é pelo menu
// do navegador.
//
// Sem ficha fora do mapa ele NÃO posta: o servidor recusaria com "escolha ao
// menos um", e uma recusa no rodapé em resposta a um clique direito parece
// defeito. Silêncio é a resposta certa para "não há o que trazer".
func atalhoDasFichas(v tabuleiroView) string {
	return "evt.preventDefault(); $escolhidosdomapa = " + fichasDoDialogo +
		"; $escolhidosdomapa && (" + comandoDePorNoMapa(v) + ")"
}
