package table

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"

	"github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// PÔR NO MAPA — ver a linha do GLOSSARY.
//
// O NOME não é `bringParty` porque esse já existe neste pacote e leva o grupo
// para a FILA. Mesmo verbo, destinos diferentes: é a colisão que a linha do
// glossário nasceu para prender.

// candidatoAoMapa é uma linha do diálogo — um combatente da fila e o que a tela
// precisa saber sobre ele.
type candidatoAoMapa struct {
	ID   string
	Name string
	// Sheet responde "é ficha de jogador ou é NPC?" (`type == "character"`), que
	// é o predicado com que o SERVIDOR escolhe o lado do mapa. Usar o mesmo aqui
	// é o que faz o atalho pôr as peças exatamente na fileira do grupo — ver a
	// colisão C4 do GLOSSARY.
	Sheet bool
	// OnBoard: já tem peça. A linha continua aparecendo, marcada e travada, em vez
	// de sumir: esconder faria o mestre procurar um nome que ele acabou de ver na
	// fila, e trazer de novo não faria nada de qualquer forma.
	OnBoard bool
}

// MapCandidates lista a fila com quem já está no mapa marcado.
//
// A ordem é a da FILA e não alfabética: é a ordem em que o mestre acabou de ler
// os nomes na tela ao lado, e reordenar aqui faria ele procurar duas vezes.
func MapCandidates(b *board.BoardState, st *live.SessionRuntimeState) []candidatoAoMapa {
	if b == nil || st == nil {
		return nil
	}
	hasToken := map[string]bool{}
	for i := range b.Tokens {
		if id := b.Tokens[i].EntryID; id != nil {
			hasToken[*id] = true
		}
	}
	list := make([]candidatoAoMapa, 0, len(st.Initiative))
	for _, entry := range st.Initiative {
		list = append(list, candidatoAoMapa{
			ID:      entry.ID,
			Name:    entry.Label,
			Sheet:   entry.Type == "character",
			OnBoard: hasToken[entry.ID],
		})
	}
	return list
}

// MapOutsideSheets são os ids que o atalho e a abertura do diálogo escolhem.
//
// Devolve uma LISTA e não um conjunto porque ela vai virar texto numa expressão
// do navegador — e a ordem estável é o que faz duas aberturas seguidas do
// diálogo desenharem a mesma coisa.
func MapOutsideSheets(candidates []candidatoAoMapa) []string {
	var ids []string
	for _, c := range candidates {
		if c.Sheet && !c.OnBoard {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// poeNoMapa faz nascer as peças escolhidas.
//
// Duas mutações e não uma: o `Populate` cria as peças e o `SetSpeeds` grava o
// ORÇAMENTO de movimento delas. Sem o segundo a peça nasce no mapa sem
// deslocamento, o alcance não acende e o jogador vê uma peça que não anda — um
// meio-recurso que ninguém reporta porque parece regra.
func poeNoMapa(st Scene, c commandCtx) (*board.BoardState, error) {
	chosen, err := escolhidosDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	board, err := st.deps.Boards().Populate(
		c.R.Context(), c.SessionID, c.BoardID, state, chosen,
	)
	if err != nil {
		return board, err
	}
	if speeds := st.speedsForBoard(c.R.Context(), board); len(speeds) > 0 {
		// O erro do deslocamento NÃO derruba o comando: as peças já nasceram e a
		// mesa precisa vê-las. Devolver erro aqui deixaria o mestre achando que
		// nada aconteceu sobre um mapa que mudou.
		if withSpeed, err := st.deps.Boards().SetSpeeds(c.R.Context(), c.SessionID, c.BoardID, speeds); err == nil {
			board = withSpeed
		}
	}
	return board, nil
}

// escolhidosDosSinais lê a escolha do diálogo.
//
// O sinal é UMA string com os ids separados por vírgula, e ela é segura porque
// id de combatente é UUID (`live.NewUUID`) — não há vírgula dentro de um id
// para partir a lista no meio. Um sinal por candidato seria um sinal por nome na
// fila, criados e destruídos a cada remendo da cena.
//
// VAZIO É ERRO, e a distinção importa: `EntrySelection` nil significa TODAS. Se
// a leitura falha ou ninguém foi escolhido, o comando recusa em vez de cair no
// "traz todo mundo" — o vilão do terceiro turno não vai para o mapa por causa de
// um sinal perdido.
func escolhidosDosSinais(r *http.Request) (board.EntrySelection, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var signals struct {
		Chosen string `json:"map_selection"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return nil, fmt.Errorf("não entendi quem pôr no mapa: %v", err)
	}
	choice := board.EntrySelection{}
	for _, id := range strings.Split(signals.Chosen, ",") {
		if id = strings.TrimSpace(id); id != "" {
			choice[id] = true
		}
	}
	if len(choice) == 0 {
		return nil, errors.New("escolha ao menos um combatente para pôr no mapa")
	}
	return choice, nil
}

// ---------------------------------------------------------------------------
// As expressões que o navegador roda. Elas moram aqui e não no `.templ` para o
// desenho ficar legível — é a mesma divisão do `takesBrush`.

// listaDeIDs escreve os ids do jeito que o sinal os guarda.
func listaDeIDs(ids []string) string { return strings.Join(ids, ",") }

// dialogSheets lê no DOM os ids das fichas que ainda não têm peça.
//
// LER O DIÁLOGO em vez de escrever os ids no botão não é preferência de estilo:
// o botão mora na região do MAPA, e id de combatente é dado da FILA. Embutido
// ali, qualquer mudança na fila mudaria o HTML do mapa e o remendo trocaria a
// peça debaixo do dedo do mestre no meio do arrasto — é o que o
// `TestATrackerChangeDoesNotPatchTheMap` prende.
const dialogSheets = "[...document.querySelectorAll('#populate [data-ficha]')]" +
	".map((e) => e.dataset.id).join(',')"

// openMap recomeça a escolha no padrão SEGURO e abre o diálogo.
//
// Cada abertura reescreve o sinal em vez de continuar de onde parou, e isso não
// é limpeza: o rascunho da vez anterior pode ter o vilão marcado, e um diálogo
// que lembra a escolha de dois minutos atrás põe a emboscada no mapa com um
// clique em "Pôr no mapa" que o mestre acha que está confirmando outra coisa.
func openMap() string {
	return "$map_selection = " + dialogSheets +
		"; document.getElementById('populate').showModal()"
}

// toggleMap liga ou desliga um id na escolha.
//
// O `filter(Boolean)` é o que impede a vírgula solta: sem ele, desmarcar o único
// escolhido deixaria a string `""` virar `[""]` ao voltar, e o servidor receberia
// um id vazio.
func toggleMap(id string) string {
	return fmt.Sprintf(
		"$map_selection = ($map_selection.split(',').filter(Boolean).includes(%q)"+
			" ? $map_selection.split(',').filter((v) => v && v !== %q)"+
			" : [...$map_selection.split(',').filter(Boolean), %q]).join(',')",
		id, id, id,
	)
}

// estaEscolhido é a pergunta que pinta o crachá.
func estaEscolhido(id string) string {
	return fmt.Sprintf("$map_selection.split(',').includes(%q)", id)
}

// mapCommand posta a escolha.
func mapCommand(v BoardView) string {
	return sceneBoardCommand(v, "pecas")
}

// sheetsShortcut é o clique DIREITO: põe só as fichas, sem diálogo.
//
// O gesto nunca é o único caminho — abrir o diálogo e confirmar faz exatamente
// isto, porque a abertura marca as mesmas fichas. O `preventDefault` é pelo menu
// do navegador.
//
// Sem ficha fora do mapa ele NÃO posta: o servidor recusaria com "escolha ao
// menos um", e uma recusa no rodapé em resposta a um clique direito parece
// defeito. Silêncio é a resposta certa para "não há o que trazer".
func sheetsShortcut(v BoardView) string {
	return "evt.preventDefault(); $map_selection = " + dialogSheets +
		"; $map_selection && (" + mapCommand(v) + ")"
}

// A PEÇA AVULSA — a porta, o baú, o barril.
//
// É o outro caminho de criar peça: o `poeNoMapa` ITERA A INICIATIVA, então por
// ele só nasce peça para quem já é combatente, e o GLOSSARY promete na linha de
// `peça` que "uma peça pode existir sem linha na fila".
//
// A POSIÇÃO VEM DO CORPO, junto com os sinais do desenho, e não do CAMINHO: o
// `loosePieceSignals` logo abaixo lê os dois do mesmo corpo, porque o
// `ReadSignals` o consome inteiro e não há segunda leitura.
func newLoosePiece(st Scene, c commandCtx) (*board.BoardState, error) {
	drawing, square, err := loosePieceSignals(c.R)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().AddToken(c.R.Context(), c.SessionID, c.BoardID, board.BoardToken{
		Label: drawing.Name, Kind: drawing.Appearance, Footprint: drawing.Size,
		X: square.X, Y: square.Y,
	})
}

// loosePieceDraft é o que o mestre escolhe na tira: o nome, o tamanho e a
// aparência.
type loosePieceDraft struct {
	Name       string
	Size       int
	Appearance string
}

// loosePieceSignals lê a tira e RECUSA o que não serve.
//
// O nome é obrigatório porque a peça inteira se identifica por ele: o monogram
// sai dele, o `aria-label` sai dele, e "eu ataco o quê?" não tem resposta sem
// ele. Uma peça sem nome nasceria muda no mapa e no leitor de tela.
//
// O tamanho é o do livro (p107, Tab. 1-21): 1 é Médio, 2 é Grande, 3 é Enorme, 6
// é Colossal. Fora dessa lista não é tamanho de criatura nenhuma, e um 40
// digitado encheria a tela de uma peça só.
func loosePieceSignals(r *http.Request) (loosePieceDraft, engine.Square, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	// UMA leitura e um struct só, porque o CORPO NÃO SE LÊ DUAS VEZES: o
	// `ReadSignals` do datastar-go copia `r.Body` inteiro num buffer, e um
	// segundo leitor pega vazio.
	var signals struct {
		Name       string             `json:"new_token_name"`
		Size       int                `json:"new_token_size"`
		Appearance string             `json:"new_token_look"`
		Square     struct{ X, Y int } `json:"from"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return loosePieceDraft{}, engine.Square{}, fmt.Errorf("não entendi a peça: %v", err)
	}
	square := engine.Square{X: signals.Square.X, Y: signals.Square.Y}
	name := strings.TrimSpace(signals.Name)
	if name == "" {
		return loosePieceDraft{}, square, errors.New("dê um nome à peça: é ele que aparece no mapa e no laço")
	}
	if !footprintsDaCasa[signals.Size] {
		return loosePieceDraft{}, square, fmt.Errorf(
			"tamanho %d não é de criatura nenhuma; o livro tem 1 (Médio), 2 (Grande), 3 (Enorme) e 6 (Colossal, p107)",
			signals.Size)
	}
	if !aparenciasDaPeca[signals.Appearance] {
		return loosePieceDraft{}, square, fmt.Errorf(
			"aparência %q não existe; a peça avulsa é objeto ou cenário", signals.Appearance)
	}
	return loosePieceDraft{Name: name, Size: signals.Size, Appearance: signals.Appearance}, square, nil
}

// footprintsDaCasa são os lados que a Tabela 1-21 produz (p107).
//
// Lista e não faixa: 4 e 5 não são tamanho de nada, e aceitá-los desenharia uma
// peça que o livro não tem. O `FootprintForSize` do motor produz exatamente
// estes quatro.
var footprintsDaCasa = map[int]bool{1: true, 2: true, 3: true, 6: true}

// aparenciasDaPeca são as duas que a peça avulsa pode ter.
//
// `character` fica de FORA de propósito: a peça de ficha nasce ligada a um
// personagem pelo `Populate`, e deixar o mestre desenhar uma "ficha" solta
// criaria uma peça que PARECE de jogador e não tem ninguém atrás dela.
var aparenciasDaPeca = map[string]bool{"object": true, "npc": true}

// speedsForBoard mede o deslocamento das peças de personagem que ainda não têm
// um. SÓ AS QUE FALTAM: recomputar a ficha de todo mundo a cada "trazer o grupo"
// seria pagar caro por um número que não muda sozinho.
//
// Ela é da CENA e não de um caso de uso (ALE-344): não autoriza nada, não grava
// nada e não decide nada — é a conta que o desenho da prévia de movimento pede,
// montada com o `Queries` e o `Catalogs` que a cena já recebe, como a Defesa do
// Grupo. A REGRA é do motor (`engine.SquaresForDisplacement`).
func (s Scene) speedsForBoard(ctx context.Context, boardState *board.BoardState) map[string]int {
	squares := map[string]int{}
	if boardState == nil {
		return squares
	}
	for _, token := range boardState.Tokens {
		if token.CharacterID == nil || token.SpeedSquares > 0 {
			continue
		}
		if n := s.speedSquaresOf(ctx, *token.CharacterID); n > 0 {
			squares[token.ID] = n
		}
	}
	return squares
}

// speedSquaresOf converte o deslocamento da ficha computada em quadrados.
//
// Falha em silêncio de propósito: a peça sem deslocamento medido desenha sem a
// prévia, e derrubar o gesto inteiro por causa de um número deixaria o mestre
// sem as peças que já nasceram.
func (s Scene) speedSquaresOf(ctx context.Context, characterID int64) int {
	row, err := s.deps.Queries().GetCharacter(ctx, characterID)
	if err != nil {
		return 0
	}
	character, err := sheet.LoadAndCompute(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		log.Printf("tabuleiro: ficha do personagem %d não computada (%v)", characterID, err)
		return 0
	}
	return engine.SquaresForDisplacement(float64(character.Displacement.Total))
}
