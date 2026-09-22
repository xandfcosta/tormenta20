package table

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"t20engine/app"
	"t20engine/app/initiative"
	"t20engine/domain/book"
	"t20engine/serve/web/master"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/creature"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
)

// OS NPCs DA CAMPANHA.
//
// O conceito é este: INICIATIVA NÃO É LISTA DE COMBATENTES. O
// taverneiro que não briga, o chefe da semana que vem, o capitão que já
// apareceu duas vezes — nenhum deles cabia na fila, e a fila esvazia ao fim da
// cena. O elenco é da CAMPANHA e não da sessão, porque "os NPCs voltam semana
// que vem" só é verdade assim.
//
// O CAMINHO PRINCIPAL É A CÓPIA: a maioria dos NPCs nasce
// como um verbete do bestiário com dois ou três números mexidos. Por isso o
// gesto de guardar mora DENTRO do painel do bestiário da Mesa, que é onde o
// mestre já está quando tem a ideia — em vez de um seletor próprio que o faria
// procurar de novo o que já estava na tela.
//
// Criar do zero é a exceção, e é o mesmo formulário com a semente em branco.

func (s Scene) RoutesNpc(r chi.Router) {
	base := sessionPattern + "/elenco/npc"
	r.Post(base+"/do-verbete", s.gmCommand(saveEntryCast))
	r.Post(base+"/{npcId}/na-fila", s.gmCommand(putNpcTracker))
	r.Post(base+"/{npcId}/apagar", s.gmCommand(eraseNpc))
}

// npcSignals é o que o painel manda ao guardar uma cópia.
//
// Nomes em `snake_case` porque viram chave de atributo (`data-bind:...`), e o
// analisador de HTML minuscula chave — caixa alta ali chega minúscula e liga um
// sinal NOVO, com o servidor lendo o antigo para sempre vazio.
type npcSignals struct {
	Creature string `json:"creature"`
	Name     string `json:"npc_name"`
}

// saveEntryCast copia um verbete do livro para o elenco da campanha.
//
// O NOME PODE VIR VAZIO, e aí é o do livro: guardar "Ogro" como "Ogro" é o caso
// mais comum, e obrigar a digitar um nome faria o mestre repetir o que a tela
// já mostra. Quem quiser "Ogro Capitão" escreve.
func saveEntryCast(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	c.R.Body = http.MaxBytesReader(nil, c.R.Body, 1<<20)
	var signals npcSignals
	if err := datastar.ReadSignals(c.R, &signals); err != nil {
		return nil, fmt.Errorf("não entendi o pedido: %v", err)
	}
	v := book.EntryByID(signals.Creature)
	if v == nil {
		return nil, fmt.Errorf("criatura %q não está no bestiário", signals.Creature)
	}
	name := strings.TrimSpace(signals.Name)
	if name == "" {
		name = v.Name
	}
	// A CENA monta o bloco a partir do verbete; validar, normalizar e gravar são
	// do caso de uso, porque o editor faz as mesmas três com um bloco de outra
	// origem (ALE-353).
	if _, err := st.cast.Save(
		c.R.Context(), st.callerOf(c.R), c.CampaignID, name, master.CopyOfEntry(*v),
	); err != nil {
		return nil, castRefusal(err, name)
	}
	// O ELENCO NÃO É ESTADO DE SESSÃO: guardar um NPC não muda a fila nem o
	// mapa. Devolver o estado mesmo assim é o que faz a cena ser redesenhada
	// com a lista nova — o `gmCommand` remenda todas as regiões a partir
	// dele, e sem isso o painel só mostraria o NPC no próximo F5.
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}

// campaignNpc lê o NPC conferindo que ele é DESTA campanha.
//
// A conferência não é zelo: o id vem do CAMINHO, e caminho é digitável. Sem
// ela, o mestre de uma mesa alcançaria o elenco de outra — e o elenco guarda a
// preparação da campanha, que é o material mais privado que o mestre tem.
func (s Scene) campaignNpc(c commandCtx) (sqlcgen.CampaignCreature, creature.Block, error) {
	id, err := strconv.ParseInt(chi.URLParam(c.R, "npcId"), 10, 64)
	if err != nil {
		return sqlcgen.CampaignCreature{}, creature.Block{}, fmt.Errorf("npc inválido: %q", chi.URLParam(c.R, "npcId"))
	}
	return s.idCampaignNpc(c, id)
}

// idCampaignNpc é o mesmo com o id já lido, para quem não o tem no caminho.
//
// O editor precisa dele porque lá o id vem do RASCUNHO — o formulário sabe quem
// está editando —, e a conferência de campanha tem de ser a MESMA. Duas cópias
// dariam duas travas, e a que envelhecesse seria a de menos uso — hoje ela mora
// no `campaign.Cast`, e isto aqui é só a tradução da recusa para FRASE, que é o
// que o Datastar precisa (ALE-353).
func (s Scene) idCampaignNpc(c commandCtx, id int64) (sqlcgen.CampaignCreature, creature.Block, error) {
	row, block, err := s.cast.Block(c.R.Context(), s.callerOf(c.R), c.CampaignID, id)
	if err != nil {
		return sqlcgen.CampaignCreature{}, creature.Block{}, castRefusal(err, fmt.Sprintf("o npc %d", id))
	}
	return row, block, nil
}

// castRefusal traduz a recusa TIPADA do caso de uso na FRASE que a cena mostra.
//
// A Mesa devolve recusa como CONTEÚDO em 200 — o Datastar descarta o corpo de
// um 4xx —, e essa é a exceção declarada no `engine-go/CLAUDE.md`. O que muda é
// de onde a frase vem: a decisão é do caso de uso e a redação é daqui.
func castRefusal(err error, what string) error {
	switch {
	case err == nil:
		// SUCESSO tem de atravessar: os chamadores a usam como último `return`,
		// e um `nil` virando erro faria o gesto que DEU CERTO desenhar recusa.
		return nil
	case errors.Is(err, app.ErrNotFound):
		return fmt.Errorf("%s não existe", what)
	case errors.Is(err, app.ErrForbidden):
		return fmt.Errorf("%s não é desta campanha", what)
	case errors.Is(err, app.ErrRefused):
		return err
	}
	return fmt.Errorf("não deu para mexer em %s: %v", what, err)
}

// putNpcTracker traz um NPC guardado para o combate.
//
// Os PV vêm do BLOCO e não de um campo da tela: o bloco é a ficha daquele NPC, e
// digitar o PV de novo ao trazê-lo seria pedir duas vezes o mesmo número — com
// a segunda podendo discordar da primeira.
func putNpcTracker(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	row, block, err := st.campaignNpc(c)
	if err != nil {
		return nil, err
	}
	novo := live.CombatantDraft{
		Label: row.Name, Initiative: block.Initiative, HP: int64(block.HP), Kind: "npc",
	}
	if err := live.ValidateCombatantDraft(novo); err != nil {
		return nil, err
	}
	// `creatureId` liga a LINHA ao bloco guardado, e é o que faz o olho da fila
	// abrir a ficha certa. É o mesmo campo que o `monsterId` do bestiário usa
	// para apontar o verbete — um diz "veio do livro", o outro "é do elenco".
	init, pv, blockID := int64(block.Initiative), int64(block.HP), row.ID
	entry, err := st.queue.Roster().Entry(c.R.Context(), app.Caller{ID: c.User}, c.CampaignID,
		initiative.EntryRequest{
			Label: row.Name, Initiative: &init, Kind: "npc",
			HpCurrent: &pv, HpMax: &pv, CreatureID: &blockID,
		})
	if err != nil {
		state, stateErr := st.deps.Sessions().State(c.R.Context(), c.SessionID)
		if stateErr != nil {
			return nil, stateErr
		}
		return state, err
	}
	return st.deps.Sessions().AddInitiativeEntry(c.SessionID, entry)
}

// eraseNpc tira o NPC do elenco da campanha.
//
// Não mexe na FILA: uma linha já posta continua na cena. Apagar o NPC do elenco
// e a linha do combate são dois gestos porque respondem a duas perguntas — "ele
// não volta mais" e "ele saiu desta cena" —, e juntá-los faria o mestre perder
// o combatente em curso ao arrumar a preparação.
func eraseNpc(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	row, _, err := st.campaignNpc(c)
	if err != nil {
		return nil, err
	}
	if _, err := st.cast.Erase(c.R.Context(), st.callerOf(c.R), c.CampaignID, row.ID); err != nil {
		return nil, castRefusal(err, strconv.Quote(row.Name))
	}
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}

// castNpc é um NPC guardado, já com o que a lista mostra.
type castNpc struct {
	ID   int64
	Name string
	// Summary é a linha do livro: "ND 3 · Humanoide Médio · PV 30 · Defesa 15".
	Summary string
	// FromBook diz de qual verbete ele foi copiado, vazio quando escrito do
	// zero. A tela usa para dizer "veio do Ogro" — o mestre reconhece a origem
	// do que ele mesmo renomeou.
	FromBook string
	PV       int
}

// CampaignCast lê os NPCs guardados.
//
// Exportada porque a BANCADA do hospedeiro a chama: dois casos provam que o NPC
// gravado aparece no elenco e some quando é apagado, e este pacote não tem
// banco. Mesma direção do `characters.Load`.
//
// Um bloco ilegível NÃO derruba a lista: ele entra com o resumo vazio, porque
// perder o elenco inteiro por causa de um JSON estragado seria trocar um
// problema pequeno por um grande — e o mestre precisa poder APAGAR o estragado.
func (s Scene) CampaignCast(ctx context.Context, campaignID int64) []castNpc {
	rows, err := s.cast.List(ctx, campaignID)
	if err != nil {
		return nil
	}
	outside := make([]castNpc, 0, len(rows))
	for _, l := range rows {
		npc := castNpc{ID: l.ID, Name: l.Name}
		var block creature.Block
		if err := json.Unmarshal([]byte(l.Block), &block); err == nil {
			npc.Summary = resumoDoBloco(block)
			npc.FromBook = block.SourceMonsterID
			npc.PV = block.HP
		}
		outside = append(outside, npc)
	}
	return outside
}

// resumoDoBloco é a linha de identidade do livro, na ordem em que ele escreve.
func resumoDoBloco(b creature.Block) string {
	// As MESMAS funções que o bestiário usa para a linha dele. Um segundo par de
	// rótulos faria o mesmo Ogro ser "Humanoide" numa tela e "humanoid" na
	// outra — e o mestre não teria como saber qual das duas está certa.
	parts := []string{
		"ND " + book.CRWritten(b.ND),
		book.TypeName(b.Kind) + " " + book.SizeName(b.Size),
		"PV " + strconv.Itoa(b.HP),
		"Defesa " + strconv.Itoa(b.Defense),
	}
	return strings.Join(parts, " · ")
}
