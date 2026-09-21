package boards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"t20engine/domain/board"
	"t20engine/infra/db/sqlcgen"
)

// O RASCUNHO DE LUGAR: montar a próxima cena FORA da sessão — preparar a sessão
// de sábado na quinta-feira, sem ninguém conectado. A cortina resolve o outro
// tempo, o de montar a cripta enquanto a mesa olha a taverna; os dois convivem,
// e o GLOSSARY registra a linha entre eles.
//
// # Por que não há um estado do rascunho em memória
//
// Porque não há o que guardar entre dois gestos. O tabuleiro vivo mora em
// memória por três razões que o rascunho não tem: a mesa inteira lê o mesmo
// estado a cada quadro do SSE, gravar é assíncrono, e a versão precisa subir
// para o cliente reconhecer o que é mais novo. Aqui há um leitor só, o gesto é
// humano, e a próxima leitura vem do disco de qualquer jeito.
//
// Então cada gesto LÊ a cena guardada, aplica a MESMA função pura que a mesa
// aplica (`board.AddToken`, `board.PaintTerrain`, `board.RemoveMarker`…) e grava de volta. Sem
// cache, sem despejo, sem uma segunda verdade sobre onde as peças estão.

// EditPlace roda uma mutação sobre a cena guardada de um lugar.
//
// A gravação passa pelo `SavePlaceScene`, então a conferência dele vale para
// todo gesto: a mutação é pura e não sabe do teto de peças nem da coordenada
// sã. Uma recusa deixa o acervo INTACTO, porque quem escreve é a gravação e ela
// não chega a acontecer.
func (bs *Store) EditPlace(
	ctx context.Context, campaignID, placeID int64, fn func(*board.BoardState) error,
) (*board.BoardState, error) {
	scene, err := bs.PlaceScene(ctx, campaignID, placeID)
	if err != nil {
		return nil, err
	}
	if err := bs.refusesIfOnATable(ctx, campaignID, scene.Place); err != nil {
		return nil, err
	}
	if err := fn(scene); err != nil {
		return nil, err
	}
	if err := bs.SavePlaceScene(ctx, campaignID, placeID, scene); err != nil {
		return nil, err
	}
	return scene, nil
}

// MaxPlaceNameLength é o teto do nome de um lugar, em LETRAS e não em bytes:
// "Câmara Mortuária de Thwor" tem acento, e contar bytes cortaria um nome mais
// curto do que o número promete.
//
// O teto existe porque o nome é a IDENTIDADE do lugar e ele é lido em coluna,
// numa lista — um nome de mil letras não é um lugar, é um parágrafo colado no
// campo errado, e ele quebra a linha de todos os vizinhos. Sessenta é o mesmo
// teto que o nome da peça já usa no diálogo de editar.
const MaxPlaceNameLength = 60

// NewID cunha um id de peça ou de marcador, com o mesmo cunho da mesa.
//
// Exposto para o RASCUNHO, e é a única coisa que ele precisa do store além do
// `EditPlace`: as mutações dele são as funções PURAS deste pacote, e elas
// recebem o cunho de fora porque o servidor é quem numera — dois clientes
// duplicando ao mesmo tempo não podem inventar o mesmo "Zumbi 3".
//
// Um cunho próprio do rascunho seria uma segunda política de identidade sobre as
// mesmas peças, e elas se encontram: a cena montada aqui vai para a mesa.
func (bs *Store) NewID() string { return bs.newID() }

// refusesIfOnATable recusa o rascunho do lugar que está ABERTO numa mesa.
//
// Seriam duas verdades sobre onde as peças estão, e a de fora perderia em
// SILÊNCIO: encerrar a aba chama o `Archive`, que sobrescreve o lugar de mesmo
// nome — a noite de trabalho no rascunho sumiria sem uma linha na tela.
//
// EM QUALQUER SESSÃO DA CAMPANHA, e não só na que está ativa: uma sessão
// ENCERRADA guarda os tabuleiros dela — o `EndSession` não toca em `open_boards`
// —, e reabri-la os traz de volta. Fechar um deles depois chamaria o `Archive`
// sobre um lugar montado semanas antes.
//
// Pelo NOME e não pelo id, porque é o nome que identifica o lugar dentro da
// campanha — é assim que o `Archive` decide se sobrescreve, e é a mesma conta
// que o acervo já faz para escrever "nesta mesa agora".
//
// DAS DUAS FONTES, e nenhuma delas basta sozinha — foi medido nas duas direções:
//
//   - só a MEMÓRIA não vê o tabuleiro aberto ontem, numa sessão que este
//     processo nunca hidratou. Depois de um `docker compose up` o mapa está
//     vazio e a trava passaria a deixar tudo montar.
//   - só o BANCO não vê o tabuleiro que acabou de ser aberto: a gravação é
//     ASSÍNCRONA (ver `persistBoardAndWarn`), e entre o `Open` e o `Persist` a
//     tabela ainda não sabe dele.
func (bs *Store) refusesIfOnATable(ctx context.Context, campaignID int64, name string) error {
	if session := bs.sessionShowingLocked(name); session != 0 {
		return placeOnATable(name)
	}
	open, err := bs.q.ListOpenBoardsOfCampaign(ctx, campaignID)
	if err != nil {
		// RECUSA em vez de deixar passar: o que está em jogo é o trabalho do
		// mestre, e o modo de falha do "deixa passar" é ele montar uma cripta
		// inteira que o `Archive` apaga depois. Um erro na tela custa um clique;
		// o silêncio custa a noite.
		return fmt.Errorf("não consegui conferir se %q está aberto numa mesa: %v", name, err)
	}
	for _, isOpen := range open {
		var scene board.BoardState
		if err := json.Unmarshal([]byte(isOpen.State), &scene); err != nil {
			// Um blob quebrado não pode virar "pode montar": ele é justamente o
			// tabuleiro sobre o qual não se sabe nada.
			return fmt.Errorf("o tabuleiro %s da sessão %d está ilegível; não dá para saber se é %q",
				isOpen.Boardid, isOpen.Sessionid, name)
		}
		if scene.Place == name {
			return placeOnATable(name)
		}
	}
	return nil
}

// PlacesOnATable diz, para cada NOME de lugar aberto numa mesa da campanha, a
// sessão que o mostra.
//
// Ela é a irmã de leitura do `refusesIfOnATable`, e as duas leem as MESMAS duas
// fontes pela mesma razão: a lista da crônica escreve "nesta mesa agora" ao lado
// do lugar, e ela tem de dizer a mesma coisa que a trava vai aplicar. Duas
// varreduras diferentes é a tela oferecendo "Montar" num lugar que o servidor
// recusa — o clique que não faz nada e não diz nada.
//
// Mapa por NOME e não por id, porque é assim que o `Archive` identifica o lugar,
// e é a chave que faz uma cena aberta do zero com o nome de um lugar guardado
// contar como aquele lugar.
func (bs *Store) PlacesOnATable(ctx context.Context, campaignID int64) map[string]int64 {
	onTable := map[string]int64{}
	open, err := bs.q.ListOpenBoardsOfCampaign(ctx, campaignID)
	if err != nil {
		// A LISTA SEGUE sem a marca, ao contrário da trava, e a assimetria é
		// deliberada: aqui o custo do erro é oferecer um botão que o servidor
		// recusa com uma frase; lá é o mestre perder a noite de trabalho. A trava
		// erra para o lado seguro, a tela erra para o lado que fala.
		log.Printf("campaign %d: falha ao listar os tabuleiros abertos (%v)", campaignID, err)
	}
	for _, isOpen := range open {
		var scene board.BoardState
		if err := json.Unmarshal([]byte(isOpen.State), &scene); err != nil {
			continue
		}
		if scene.Place != "" {
			onTable[scene.Place] = isOpen.Sessionid
		}
	}
	// A MEMÓRIA por cima do disco, e não o contrário: o tabuleiro aberto agora
	// ainda não foi gravado, e é justamente o que a pessoa acabou de fazer.
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	for sessionID, alive := range bs.boards {
		for _, b := range alive {
			if b.Place != "" {
				onTable[b.Place] = sessionID
			}
		}
	}
	return onTable
}

// placeOnATable é a frase da recusa, escrita UMA vez.
//
// As duas fontes acima chegam à mesma conclusão, e duas cópias da frase é como
// uma delas passa a dizer outra coisa — quem lê a tela não sabe nem deve saber
// se quem pegou foi a memória ou o disco.
func placeOnATable(name string) error {
	return fmt.Errorf(
		"%q está aberto numa mesa agora: encerre a aba antes de montar o rascunho, senão o que você montar aqui some quando ela for encerrada",
		name)
}

// sessionShowingLocked procura o lugar entre os tabuleiros VIVOS deste processo,
// e devolve a sessão que o mostra (zero quando nenhuma).
//
// Varre todas as sessões em memória e não uma: o rascunho não sabe em qual mesa
// o lugar poderia estar, e é justamente essa a pergunta. O custo é um laço sobre
// as sessões hidratadas, sem I/O nenhum, debaixo da trava que já protege o mapa.
func (bs *Store) sessionShowingLocked(name string) int64 {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	for sessionID, open := range bs.boards {
		for _, b := range open {
			if b.Place == name {
				return sessionID
			}
		}
	}
	return 0
}

// NewPlace devolve o lugar em que o mestre vai montar a cena, criando-o vazio
// quando ele ainda não existe.
//
// DEVOLVE O EXISTENTE em vez de recusar o nome repetido, e não é conveniência: o
// nome É a identidade do lugar dentro da campanha (ver o `Archive`, que
// sobrescreve por nome). Recusar diria que há dois conceitos onde há um, e criar
// um segundo deixaria duas tavernas quase iguais no acervo — que é exatamente o
// que o `Archive` existe para não fazer.
//
// O `terrain` só vale para o lugar que NASCE: pedi-lo de novo sobre uma cena
// montada repintaria o chão dela por baixo do pano, e quem quer trocar o chão
// tem o gesto do próprio rascunho para isso.
func (bs *Store) NewPlace(ctx context.Context, campaignID int64, name, terrain string) (board.Place, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return board.Place{}, errors.New("dê um nome ao lugar: é ele que identifica a cena no acervo, e é por ele que encerrar o tabuleiro reconhece qual lugar sobrescrever")
	}
	if len([]rune(name)) > MaxPlaceNameLength {
		return board.Place{}, fmt.Errorf("o nome do lugar tem %d letras (máximo %d)", len([]rune(name)), MaxPlaceNameLength)
	}
	if existing, err := bs.q.FindCampaignPlaceByName(ctx, sqlcgen.FindCampaignPlaceByNameParams{
		Campaignid: campaignID,
		Name:       name,
	}); err == nil {
		return board.Place{
			ID: existing.ID, Name: existing.Name,
			Tokens: countTokens(existing.State), UpdatedAt: existing.Updatedat,
		}, nil
	}
	// A cena nasce com a versão em 1 e as peças em fatia VAZIA, como a do
	// `Open`: `null` no JSON derruba quem indexa `tokens.length`, e é o mesmo
	// cuidado que o `storedScene` toma na volta.
	blob, err := json.Marshal(&board.BoardState{
		// O chão passa pelo catálogo ANTES de ser gravado: quem cria era a porta
		// sem guarda, e um id que a folha não pinta vira mapa sem textura.
		Version: 1, Place: name, Terrain: board.KnownGround(terrain), Tokens: []board.BoardToken{},
	})
	if err != nil {
		return board.Place{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := bs.q.SaveCampaignPlace(ctx, sqlcgen.SaveCampaignPlaceParams{
		Campaignid: campaignID, Name: name, State: string(blob),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		return board.Place{}, err
	}
	return board.Place{ID: row.ID, Name: row.Name, Tokens: 0, UpdatedAt: row.Updatedat}, nil
}
