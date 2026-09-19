package character

import (
	"context"
	"fmt"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// LevelClass é A REGRA do degrau de nível, e ela é UMA.
//
// O nível de um personagem é a SOMA dos níveis de classe — *"seu nível de
// personagem é a soma dos níveis de todas as suas classes"* (**p35**) —, e uma
// segunda cópia dessa conta é como duas telas passam a discordar de quanto vale
// o Guerreiro 3 / Ladino 2.
//
// As três garantias que ela carrega, e que a tela não pode reescrever:
//
//   - a classe TEM de ser do personagem — subir "Bardo" em quem não é bardo
//     criaria um nível que não existe em lugar nenhum;
//   - o TOTAL para em 20, que é onde a Tabela 1-4 termina (p35). A SOMA é a
//     conta que decide, não o campo;
//   - os POÇOS acompanham, porque PV e PM máximos derivam dos níveis de classe.
//     Gravar o nível sem sincronizar deixa a ficha com o número novo e a vida
//     velha, que é o defeito que ninguém liga ao botão que o causou.
//
// O acompanhamento dos poços é o `sheet.RefreshPools`, o MESMO que o passo de
// atributo usa: com o máximo derivado e o DANO gravado, um herói que apanhou e
// sobe de nível ganha os PV novos sem ganhar a cura — a conta do delta some
// porque o que ele deve continua sendo o que ele deve (ALE-355).
func (p Plays) LevelClass(
	ctx context.Context, row sqlcgen.Character, classe string, nivel int64,
) error {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return err
	}
	achou := false
	var total int64
	for i := range dto.Classes {
		if dto.Classes[i].ClassName == classe {
			dto.Classes[i].Level = nivel
			achou = true
		}
		total += dto.Classes[i].Level
	}
	if !achou {
		return fmt.Errorf("%s não é uma classe deste personagem", classe)
	}
	if total > 20 {
		return fmt.Errorf("as classes somariam o nível %d, e o 20º é o último (p35)", total)
	}
	if _, err := p.queries.SetCharacterClassLevel(ctx, sqlcgen.SetCharacterClassLevelParams{
		Level: nivel, CharacterId: row.ID, ClassName: classe,
	}); err != nil {
		return fmt.Errorf("gravar o nível %d de %s: %w", nivel, classe, err)
	}
	if err := p.queries.SetCharacterLevel(ctx, sqlcgen.SetCharacterLevelParams{
		Level: total, UpdatedAt: dbvalue.NowISO(), ID: row.ID,
	}); err != nil {
		return fmt.Errorf("gravar o nível %d da ficha %d: %w", total, row.ID, err)
	}
	row.Level = total
	_, err = sheet.RefreshPools(ctx, p.queries, p.catalogs, row)
	return err
}
