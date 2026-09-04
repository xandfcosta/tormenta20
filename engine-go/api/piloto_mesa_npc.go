package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"t20engine/book"
	"t20engine/web/master"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
	"t20engine/creature"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// OS NPCs DA CAMPANHA (ALE-269, superfície 6b).
//
// O conceito é o da ALE-212: **iniciativa não é lista de combatentes**. O
// taverneiro que não briga, o chefe da semana que vem, o capitão que já
// apareceu duas vezes — nenhum deles cabia na fila, e a fila esvazia ao fim da
// cena. O elenco é da CAMPANHA e não da sessão, porque "os NPCs voltam semana
// que vem" só é verdade assim.
//
// O CAMINHO PRINCIPAL É A CÓPIA (decisão do dono): a maioria dos NPCs nasce
// como um verbete do bestiário com dois ou três números mexidos. Por isso o
// gesto de guardar mora DENTRO do painel do bestiário da Mesa, que é onde o
// mestre já está quando tem a ideia — em vez de um seletor próprio que o faria
// procurar de novo o que já estava na tela.
//
// Criar do zero é a exceção, e é o mesmo formulário com a semente em branco.

func (s *Server) RoutesNpc(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/elenco/npc"
	r.Post(base+"/do-verbete", s.gmCommand(saveEntryCast))
	r.Post(base+"/{npcId}/na-fila", s.gmCommand(putNpcTracker))
	r.Post(base+"/{npcId}/apagar", s.gmCommand(eraseNpc))
}

// npcSignals é o que o painel manda ao guardar uma cópia.
//
// Nomes TODOS MINÚSCULOS porque viram chave de atributo (`data-bind:...`), e o
// analisador de HTML minuscula chave — um `data-bind:nomeDoNpc` chega como
// `nomedonpc` e liga um sinal NOVO, com o servidor lendo o antigo para sempre
// vazio.
type npcSignals struct {
	Criatura string `json:"criatura"`
	Nome     string `json:"nomedonpc"`
}

// saveEntryCast copia um verbete do livro para o elenco da campanha.
//
// O NOME PODE VIR VAZIO, e aí é o do livro: guardar "Ogro" como "Ogro" é o caso
// mais comum, e obrigar a digitar um nome faria o mestre repetir o que a tela
// já mostra. Quem quiser "Ogro Capitão" escreve.
func saveEntryCast(st *Server, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	c.R.Body = http.MaxBytesReader(nil, c.R.Body, 1<<20)
	var sinais npcSignals
	if err := datastar.ReadSignals(c.R, &sinais); err != nil {
		return nil, fmt.Errorf("não entendi o pedido: %v", err)
	}
	v := book.EntryByID(sinais.Criatura)
	if v == nil {
		return nil, fmt.Errorf("criatura %q não está no bestiário", sinais.Criatura)
	}
	nome := strings.TrimSpace(sinais.Nome)
	if nome == "" {
		nome = v.Name
	}
	bloco := master.CopyOfEntry(*v)
	if err := creature.Validate(nome, &bloco); err != nil {
		return nil, err
	}
	creature.Normalize(&bloco)
	blob, err := json.Marshal(bloco)
	if err != nil {
		return nil, fmt.Errorf("não deu para guardar o bloco de %q", nome)
	}
	agora := plataforma.NowISO()
	if _, err := st.queries.CreateCampaignCreature(c.R.Context(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: c.CampaignID, Name: nome, Block: string(blob),
		Createdat: agora, Updatedat: agora,
	}); err != nil {
		return nil, fmt.Errorf("não deu para guardar %q no elenco: %v", nome, err)
	}
	// O ELENCO NÃO É ESTADO DE SESSÃO: guardar um NPC não muda a fila nem o
	// mapa. Devolver o estado mesmo assim é o que faz a cena ser redesenhada
	// com a lista nova — o `gmCommand` remenda todas as regiões a partir
	// dele, e sem isso o painel só mostraria o NPC no próximo F5.
	return st.sessions.GetState(c.SessionID), nil
}

// campaignNpc lê o NPC conferindo que ele é DESTA campanha.
//
// A conferência não é zelo: o id vem do CAMINHO, e caminho é digitável. Sem
// ela, o mestre de uma mesa alcançaria o elenco de outra — e o elenco guarda a
// preparação da campanha, que é o material mais privado que o mestre tem.
func (s *Server) campaignNpc(c commandCtx) (sqlcgen.CampaignCreature, creature.Block, error) {
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
// dariam duas travas, e a que envelhecesse seria a de menos uso.
func (s *Server) idCampaignNpc(c commandCtx, id int64) (sqlcgen.CampaignCreature, creature.Block, error) {
	var bloco creature.Block
	linha, err := s.queries.GetCampaignCreature(c.R.Context(), id)
	if err != nil {
		return sqlcgen.CampaignCreature{}, bloco, fmt.Errorf("o npc %d não existe", id)
	}
	if linha.Campaignid != c.CampaignID {
		return sqlcgen.CampaignCreature{}, bloco, fmt.Errorf("o npc %d não é desta campanha", id)
	}
	if err := json.Unmarshal([]byte(linha.Block), &bloco); err != nil {
		return linha, bloco, fmt.Errorf("o bloco de %q está ilegível", linha.Name)
	}
	return linha, bloco, nil
}

// putNpcTracker traz um NPC guardado para o combate.
//
// Os PV vêm do BLOCO e não de um campo da tela: o bloco é a ficha daquele NPC, e
// digitar o PV de novo ao trazê-lo seria pedir duas vezes o mesmo número — com
// a segunda podendo discordar da primeira.
func putNpcTracker(st *Server, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	linha, bloco, err := st.campaignNpc(c)
	if err != nil {
		return nil, err
	}
	novo := aovivo.CombatantDraft{
		Label: linha.Name, Initiative: bloco.Iniciativa, HP: int64(bloco.HP), Kind: "npc",
	}
	if err := aovivo.ValidateCombatantDraft(novo); err != nil {
		return nil, err
	}
	// `creatureId` liga a LINHA ao bloco guardado, e é o que faz o olho da fila
	// abrir a ficha certa. É o mesmo campo que o `monsterId` do bestiário usa
	// para apontar o verbete — um diz "veio do livro", o outro "é do elenco".
	entrada, err := st.materializeEntry(c.R.Context(), c.User.ID, c.CampaignID, map[string]any{
		"label": linha.Name, "initiative": bloco.Iniciativa, "type": "npc",
		"hpCurrent": bloco.HP, "hpMax": bloco.HP, "creatureId": linha.ID,
	})
	if err != nil {
		return st.sessions.GetState(c.SessionID), err
	}
	return st.sessions.AddInitiativeEntry(c.SessionID, entrada)
}

// eraseNpc tira o NPC do elenco da campanha.
//
// Não mexe na FILA: uma linha já posta continua na cena. Apagar o NPC do elenco
// e a linha do combate são dois gestos porque respondem a duas perguntas — "ele
// não volta mais" e "ele saiu desta cena" —, e juntá-los faria o mestre perder
// o combatente em curso ao arrumar a preparação.
func eraseNpc(st *Server, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	linha, _, err := st.campaignNpc(c)
	if err != nil {
		return nil, err
	}
	if err := st.queries.DeleteCampaignCreature(c.R.Context(), linha.ID); err != nil {
		return nil, fmt.Errorf("não deu para apagar %q: %v", linha.Name, err)
	}
	return st.sessions.GetState(c.SessionID), nil
}

// castNpc é um NPC guardado, já com o que a lista mostra.
type castNpc struct {
	ID   int64
	Nome string
	// Resumo é a linha do livro: "ND 3 · Humanoide Médio · PV 30 · Defesa 15".
	Resumo string
	// DoLivro diz de qual verbete ele foi copiado, vazio quando escrito do
	// zero. A tela usa para dizer "veio do Ogro" — o mestre reconhece a origem
	// do que ele mesmo renomeou.
	DoLivro string
	PV      int
}

// campaignCast lê os NPCs guardados.
//
// Um bloco ilegível NÃO derruba a lista: ele entra com o resumo vazio, porque
// perder o elenco inteiro por causa de um JSON estragado seria trocar um
// problema pequeno por um grande — e o mestre precisa poder APAGAR o estragado.
func (s *Server) campaignCast(ctx context.Context, campaignID int64) []castNpc {
	linhas, err := s.queries.ListCampaignCreatures(ctx, campaignID)
	if err != nil {
		return nil
	}
	fora := make([]castNpc, 0, len(linhas))
	for _, l := range linhas {
		npc := castNpc{ID: l.ID, Nome: l.Name}
		var bloco creature.Block
		if err := json.Unmarshal([]byte(l.Block), &bloco); err == nil {
			npc.Resumo = resumoDoBloco(bloco)
			npc.DoLivro = bloco.SourceMonsterID
			npc.PV = bloco.HP
		}
		fora = append(fora, npc)
	}
	return fora
}

// resumoDoBloco é a linha de identidade do livro, na ordem em que ele escreve.
func resumoDoBloco(b creature.Block) string {
	// As MESMAS funções que o bestiário usa para a linha dele. Um segundo par de
	// rótulos faria o mesmo Ogro ser "Humanoide" numa tela e "humanoid" na
	// outra — e o mestre não teria como saber qual das duas está certa.
	partes := []string{
		"ND " + book.CRWritten(b.ND),
		book.TypeName(b.Tipo) + " " + book.SizeName(b.Size),
		"PV " + strconv.Itoa(b.HP),
		"Defesa " + strconv.Itoa(b.Defesa),
	}
	return strings.Join(partes, " · ")
}
