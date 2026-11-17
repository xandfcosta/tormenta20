package table

import (
	"context"
	"log"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// tableSheets são as fichas que UM desenho da mesa lê, carregadas uma vez cada.
//
// A Defesa no Grupo e o dano recebido leem a mesma ficha, e cada um a carregava
// por conta própria: com um grupo de seis, a página do mestre fazia doze cargas,
// dois terços do pedido. Vive só durante o pedido, de propósito: a ficha muda a
// cada gesto, e uma cópia que sobrevivesse a ele desenharia o número velho.
type tableSheets struct {
	ctx    context.Context
	scene  Scene
	loaded map[int64]loadedSheet
}

type loadedSheet struct {
	dto sheet.CharacterDTO
	ok  bool
}

func (s Scene) sheetsOfThisDrawing(ctx context.Context) *tableSheets {
	return &tableSheets{ctx: ctx, scene: s, loaded: map[int64]loadedSheet{}}
}

// of é a ficha carregada do personagem, ou falso quando ela não monta. A falha
// também fica guardada: quem pergunta de novo no mesmo desenho recebe o mesmo não,
// sem uma segunda tentativa que repetiria o log.
func (t *tableSheets) of(characterID int64) (sheet.CharacterDTO, bool) {
	if got, seen := t.loaded[characterID]; seen {
		return got.dto, got.ok
	}
	got := loadedSheet{}
	if t.scene.deps.Catalogs() != nil {
		if row, err := t.scene.deps.Queries().GetCharacter(t.ctx, characterID); err != nil {
			log.Printf("mesa: ler a ficha %d falhou (%v)", characterID, err)
		} else if dto, err := sheet.Load(t.ctx, t.scene.deps.Queries(), t.scene.deps.Catalogs(), row); err != nil {
			log.Printf("mesa: montar a ficha %d falhou (%v)", characterID, err)
		} else {
			got = loadedSheet{dto: dto, ok: true}
		}
	}
	t.loaded[characterID] = got
	return got.dto, got.ok
}

// memberDefense pergunta ao MOTOR, que é a mesma `ComputeSheet` da ficha.
//
// Travessão quando o motor não está de pé: a cena inteira não pode cair por
// causa de um número, e um zero seria pior — Defesa 0 é um valor plausível, e o
// mestre agiria sobre ele. É a mesma escolha que o cartão de personagem faz.
func (t *tableSheets) memberDefense(characterID int64) string {
	dto, ok := t.of(characterID)
	if !ok {
		return "—"
	}
	character, err := sheet.Compute(dto)
	if err != nil {
		return "—"
	}
	// A MESMA frase da ficha, e da mesma função: o mestre confere aqui a Defesa
	// de um jogador para decidir se o ataque acerta, e com o alvo caído o total
	// é o único número que não responde essa pergunta.
	return book.DefenseLabel(character.Defense)
}

// initiativeBonus é o bônus de iniciativa do personagem, pela regra do MOTOR
// (`engine.InitiativeTotal`) sobre a ficha deste desenho: o `initiative.Roster`
// carregaria a mesma ficha outra vez para fazer a mesma conta.
func (t *tableSheets) initiativeBonus(characterID int64) (int64, bool) {
	dto, ok := t.of(characterID)
	if !ok {
		return 0, false
	}
	character, err := sheet.Compute(dto)
	if err != nil {
		return 0, false
	}
	return int64(engine.InitiativeTotal(character)), true
}

// ownedOnTable são os personagens da mesa que são do usuário, e o primeiro deles.
//
// É tudo o que os gestos perguntam à mesa — de quem é a peça, quem ataca, de quem
// é o teste —, e por isso não carrega ficha nenhuma: a prévia do arrasto roda a
// cada casa que o dedo cruza, e montava o Grupo inteiro, com a ficha de cada
// membro, para ler só a posse. Leitura que falha deixa a posse em falso, que é o
// lado seguro: o gesto recusa com a frase dele.
func (s Scene) ownedOnTable(ctx context.Context, userID, campaignID int64) (map[int64]bool, *tableMe) {
	mine := map[int64]bool{}
	rows, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return mine, nil
	}
	var eu *tableMe
	for _, m := range rows {
		if owner, err := s.deps.Queries().GetCharacterOwner(ctx, m.Characterid); err == nil && owner == userID {
			mine[m.Characterid] = true
			if eu == nil {
				eu = &tableMe{CharacterID: m.Characterid, Name: m.Charname}
			}
		}
	}
	return mine, eu
}
