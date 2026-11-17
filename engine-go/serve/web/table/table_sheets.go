package table

import "context"

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
