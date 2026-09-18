// Package initiative é QUEM ENTRA NA FILA de combate, e com que números.
//
// Três pedidos chegam aqui, e é o mesmo caminho para os três: o jogador
// registrando o próprio d20, o mestre digitando um capanga na hora, e o verbete
// do bestiário virando linha. Uma segunda montagem divergiria no dia em que a
// primeira mudasse, e a diferença apareceria como uma linha sem ficha — o
// defeito que este caminho único existe para não repetir.
//
// # O pedido é TIPADO, e isso é a fronteira
//
// Ele era um `map[string]any` que cada chamador montava com as chaves na mão, e
// o servidor lia com `wire.IntField`. Uma chave escrita errado não dava erro:
// o campo simplesmente não chegava, e a linha nascia sem PV. Com um struct, o
// compilador cobra.
package initiative

import (
	"context"
	"fmt"
	"strings"

	"t20engine/app"
	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Combatant é quem entra na fila com ficha atrás: nome e vitais vivos.
type Combatant struct {
	CharacterID int64
	Name        string
	HpCurrent   int64
	HpMax       int64
	MpCurrent   int64
	MpMax       int64
}

// EntryRequest é o pedido de uma linha nova.
//
// Os PONTEIROS são o que distingue "não mandou" de "mandou zero", e a diferença
// tem consequência na tela: PV ausente é "sem vida registrada" e não desenha
// barra; PV zero desenharia uma barra vazia, que diz que o capanga já está
// morto.
type EntryRequest struct {
	// CharacterID presente faz a linha ser de PERSONAGEM: o nome e os vitais
	// vêm da ficha, e os campos abaixo viram sobreposição.
	CharacterID *int64
	Label       string
	Initiative  *int64
	Kind        string
	HpCurrent   *int64
	HpMax       *int64
	MpCurrent   *int64
	MpMax       *int64
	// MonsterID aponta o verbete do livro; CreatureID, o bloco do elenco da
	// campanha. Um diz "veio do livro", o outro "é do mestre".
	MonsterID  string
	CreatureID *int64
	Conditions []string
}

// Roster monta as linhas da fila.
type Roster struct {
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
}

func NewRoster(q *sqlcgen.Queries, catalogs *engine.Catalogs) Roster {
	return Roster{queries: q, catalogs: catalogs}
}

// Bonus é o bônus de Iniciativa da ficha, pelo motor.
//
// A conta é do livro e mora pura em `domain/engine`; aqui só se carrega a ficha
// e se computa.
func (r Roster) Bonus(ctx context.Context, characterID int64) (int64, error) {
	if r.catalogs == nil {
		return 0, fmt.Errorf("o catálogo de regras não está carregado")
	}
	row, err := r.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return 0, fmt.Errorf("carregar o personagem %d: %w", characterID, err)
	}
	ficha, err := sheet.LoadAndCompute(ctx, r.queries, r.catalogs, row)
	if err != nil {
		return 0, fmt.Errorf("computar a ficha do personagem %d: %w", characterID, err)
	}
	return int64(engine.InitiativeTotal(ficha)), nil
}

// SelfEntry é a linha de quem registra a PRÓPRIA iniciativa: confere o d20,
// pergunta o bônus ao motor e soma.
//
// O pedido é montado AQUI e não recebido do cliente: um `initiative` que ele
// mandasse junto venceria a conta do servidor.
func (r Roster) SelfEntry(
	ctx context.Context, quem app.Caller, campaignID, characterID, d20 int64,
) (live.InitiativeEntry, error) {
	if d20 < 1 || d20 > 20 {
		return live.InitiativeEntry{}, fmt.Errorf(
			"o d20 rolado foi %d, e um d20 vai de 1 a 20: %w", d20, app.ErrRefused)
	}
	bonus, err := r.Bonus(ctx, characterID)
	if err != nil {
		return live.InitiativeEntry{}, err
	}
	total := d20 + bonus
	return r.Entry(ctx, quem, campaignID, EntryRequest{CharacterID: &characterID, Initiative: &total})
}

// Entry resolve um pedido numa linha concreta.
func (r Roster) Entry(
	ctx context.Context, quem app.Caller, campaignID int64, pedido EntryRequest,
) (live.InitiativeEntry, error) {
	if pedido.CharacterID == nil {
		return r.npcEntry(pedido)
	}
	return r.characterEntry(ctx, quem, campaignID, pedido)
}

// npcEntry é a linha sem ficha atrás.
func (r Roster) npcEntry(pedido EntryRequest) (live.InitiativeEntry, error) {
	rotulo := strings.TrimSpace(pedido.Label)
	if rotulo == "" {
		return live.InitiativeEntry{}, fmt.Errorf("um NPC precisa de rótulo: %w", app.ErrRefused)
	}
	if pedido.Initiative == nil {
		return live.InitiativeEntry{}, fmt.Errorf("a linha precisa de iniciativa: %w", app.ErrRefused)
	}
	tipo := "npc"
	if pedido.Kind != "" {
		tipo = pedido.Kind
	}
	linha := live.InitiativeEntry{Label: rotulo, Initiative: int(*pedido.Initiative), Type: tipo}
	linha.HpCurrent, linha.HpMax = pedido.HpCurrent, pedido.HpMax
	// O id do verbete e o do bloco vêm do CLIENTE, e o servidor não os confere
	// contra o catálogo de propósito: um id desconhecido vira "sem bloco" na
	// tela, e não um erro que derruba a adição no meio do combate. Quem confere
	// o dono do bloco é a rota que o serve, e ela só responde ao mestre.
	if id := strings.TrimSpace(pedido.MonsterID); id != "" {
		linha.MonsterID = &id
	}
	if pedido.CreatureID != nil && *pedido.CreatureID > 0 {
		linha.CreatureID = pedido.CreatureID
	}
	linha.Conditions = KnownConditions(pedido.Conditions)
	return linha, nil
}

// characterEntry busca o nome e os vitais na ficha, com as sobreposições que o
// pedido trouxer.
func (r Roster) characterEntry(
	ctx context.Context, quem app.Caller, campaignID int64, pedido EntryRequest,
) (live.InitiativeEntry, error) {
	if pedido.Initiative == nil {
		return live.InitiativeEntry{}, fmt.Errorf("a linha precisa de iniciativa: %w", app.ErrRefused)
	}
	quemEntra, err := r.Combatant(ctx, quem, campaignID, *pedido.CharacterID)
	if err != nil {
		return live.InitiativeEntry{}, err
	}
	rotulo := quemEntra.Name
	if escolhido := strings.TrimSpace(pedido.Label); escolhido != "" {
		rotulo = escolhido
	}
	id := *pedido.CharacterID
	return live.InitiativeEntry{
		Label: rotulo, Initiative: int(*pedido.Initiative), Type: "character", CharacterID: &id,
		HpCurrent:  overriddenOr(pedido.HpCurrent, quemEntra.HpCurrent),
		HpMax:      overriddenOr(pedido.HpMax, quemEntra.HpMax),
		MpCurrent:  overriddenOr(pedido.MpCurrent, quemEntra.MpCurrent),
		MpMax:      overriddenOr(pedido.MpMax, quemEntra.MpMax),
		Conditions: KnownConditions(pedido.Conditions),
	}, nil
}

// overriddenOr devolve a sobreposição quando ela veio, e o valor da ficha
// quando não.
func overriddenOr(sobreposto *int64, daFicha int64) *int64 {
	if sobreposto != nil {
		return live.PtrInt64(*sobreposto)
	}
	return live.PtrInt64(daFicha)
}

// KnownConditions filtra pelo CATÁLOGO, que é onde as condições são autoradas.
//
// Uma lista escrita à mão aqui seria a segunda cópia da tabela do livro, e ela
// desvia — uma condição faltando dá 400 na hora de aplicá-la. Id desconhecido é
// descartado em silêncio de propósito: a alternativa seria derrubar a aplicação
// inteira no meio do combate por causa de um item.
func KnownConditions(pedidas []string) []string {
	fora := []string{}
	visto := map[string]bool{}
	for _, id := range pedidas {
		if id == "" || visto[id] || !catalog.IsCondition(id) {
			continue
		}
		visto[id] = true
		fora = append(fora, id)
	}
	return fora
}
