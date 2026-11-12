package sheet

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// SaveConditions grava as condições da ficha, e é o ÚNICO caminho de escrita delas (o
// `TestEveryConditionWriteGoesThroughTheFunnel` o prende).
//
// Ele existe pelo que uma condição DESFAZ ao entrar, e não pela gravação: "você reverte à forma normal
// se ficar inconsciente ou morrer" (p63). A Inconsciente chega por várias portas — o dano que zera os PV,
// a mão do jogador na ficha, a do mestre na Mesa, a API —, e uma regra escrita em cada uma seria
// esquecida na próxima. Morrer de pé também liga a Inconsciente (p236), então ela basta.
//
// A forma não volta quando a condição sai: o livro diz que ela REVERTE, e voltar é transformar de novo,
// pagando o PM. A transação é de quem chama, porque cada porta já tem a dela (ou nenhuma).
//
// @example sheet.SaveConditions(ctx, q, id, []string{"inconsciente", "sangrando"})
func SaveConditions(ctx context.Context, q *sqlcgen.Queries, characterID int64, conditions []string) error {
	if err := q.UpdateConditions(ctx, sqlcgen.UpdateConditionsParams{
		ActiveConditions: MarshalStrings(&conditions), UpdatedAt: dbvalue.NowISO(), ID: characterID,
	}); err != nil {
		return fmt.Errorf("gravar as condições da ficha %d: %w", characterID, err)
	}
	if !slices.Contains(conditions, engine.ConditionUnconscious) {
		return nil
	}
	if err := q.ClearCharacterWildShape(ctx, characterID); err != nil {
		return fmt.Errorf("reverter a forma selvagem da ficha %d, que caiu inconsciente: %w", characterID, err)
	}
	return nil
}
