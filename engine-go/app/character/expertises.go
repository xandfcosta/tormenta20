package character

import (
	"context"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS PERÍCIAS E AS PROFICIÊNCIAS DA FICHA.
//
// Os cinco gestos daqui têm a mesma forma, e é ela que os traz para esta
// camada: LEEM o que está gravado, DECIDEM, e gravam. A decisão não é
// encanamento em nenhum deles — alternar treino precisa saber o estado atual,
// remover um ofício precisa saber se a linha é do livro, e criar um precisa
// saber se o nome está livre.
//
// Até a ALE-350 eles moravam em dois lugares ao mesmo tempo: a regra do nome e
// a das proficiências no `serve/api`, e o `INSERT`/`UPDATE` na cena, logo
// depois. As duas metades andavam juntas por hábito.

// ToggleTraining liga ou desliga o treino de UMA perícia.
//
// Recebe a PERÍCIA e não o estado desejado: mandar "treinada" perde para o
// clique repetido e para a segunda aba aberta no mesmo personagem.
func (p Plays) ToggleTraining(ctx context.Context, characterID int64, nome string) error {
	// O estado ATUAL vem da lista e não do `GetExpertiseMeta`, que devolve só o
	// id e o `custom` — inverter exige saber o que está lá.
	todas, err := p.queries.ListExpertisesByCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("ler as perícias da ficha %d: %w", characterID, err)
	}
	for _, e := range todas {
		if e.Name != nome {
			continue
		}
		depois := e.Trained == 0
		if _, err := p.queries.UpdateExpertise(ctx, sqlcgen.UpdateExpertiseParams{
			Trained: dbvalue.NullBool(&depois), CharacterId: characterID, Name: nome,
		}); err != nil {
			return fmt.Errorf("gravar o treino de %q: %w", nome, err)
		}
		return nil
	}
	return fmt.Errorf("a perícia %q não é desta ficha", nome)
}

// SwapAttribute repõe a perícia em outro atributo.
func (p Plays) SwapAttribute(ctx context.Context, characterID int64, nome, atributo string) error {
	if !engine.IsAttributeKey(atributo) {
		return fmt.Errorf("%q não é um atributo: são %v", atributo, engine.AttributeKeys)
	}
	if _, err := p.queries.UpdateExpertise(ctx, sqlcgen.UpdateExpertiseParams{
		Attribute: dbvalue.NullString(&atributo), CharacterId: characterID, Name: nome,
	}); err != nil {
		return fmt.Errorf("a perícia %q não é desta ficha", nome)
	}
	return nil
}

// RemoveCraft apaga uma perícia INVENTADA pelo jogador.
//
// As do livro não se apagam, e a recusa é do SERVIDOR e não da tela: travar só
// na interface deixaria a regra sem fronteira, e quem montasse o `@post` à mão
// apagaria a Fortitude.
func (p Plays) RemoveCraft(ctx context.Context, characterID int64, nome string) error {
	meta, err := p.queries.GetExpertiseMeta(ctx, sqlcgen.GetExpertiseMetaParams{
		Characterid: characterID, Name: nome,
	})
	if err != nil {
		return fmt.Errorf("a perícia %q não é desta ficha", nome)
	}
	// A COLUNA decide, e não a lista das 29: `custom` é o que o banco guarda
	// sobre esta linha, enquanto a lista é uma opinião do código sobre o nome. As
	// duas concordam hoje; no dia em que uma perícia nova entrar no livro, a
	// coluna continua certa e a lista fica velha.
	if meta.Custom == 0 {
		return fmt.Errorf("%q é uma perícia do livro e não se remove da ficha", nome)
	}
	if err := p.queries.DeleteExpertiseByID(ctx, meta.ID); err != nil {
		return fmt.Errorf("apagar o ofício %q: %w", nome, err)
	}
	return nil
}

// AddCraft acrescenta uma perícia que o livro não tem — o saber de um ferreiro,
// a arte de um marinheiro.
//
// Ela nasce TREINADA, porque inventar um ofício e não tê-lo treinado não é um
// estado que signifique alguma coisa.
//
// # O nome repetido é decidido pelo BANCO, e não por uma leitura antes
//
// Havia uma conferência (`GetExpertiseMeta`) e, depois dela, o `INSERT` — em
// pacotes diferentes. A `UNIQUE (characterId, name)` já responde a mesma
// pergunta sem janela nenhuma entre a resposta e a escrita, então o que sobra é
// TRADUZIR a violação na frase que o jogador lê. Uma ida ao banco em vez de
// duas, e nada para os dois discordarem (ALE-350).
//
// A lista das 29 vem do `domain/sheet`, que é onde a tabela do livro mora. Ela
// estava COPIADA no `serve/api`, a dois arquivos da original — e o comentário da
// original já avisava que duas cópias divergem num acento.
func (p Plays) AddCraft(ctx context.Context, characterID int64, nome, atributo string) error {
	if nome == "" {
		return fmt.Errorf("dê um nome ao ofício")
	}
	if sheet.IsBuiltinExpertise(nome) {
		return fmt.Errorf("%q é uma perícia do livro — escolha outro nome", nome)
	}
	if !engine.IsAttributeKey(atributo) {
		return fmt.Errorf("%q não é um atributo: são %v", atributo, engine.AttributeKeys)
	}
	_, err := p.queries.CreateExpertise(ctx, sqlcgen.CreateExpertiseParams{
		Characterid: characterID, Name: nome, Attribute: atributo, Trained: 1, Custom: 1,
	})
	if db.IsUniqueViolation(err) {
		return fmt.Errorf("esta ficha já tem %q", nome)
	}
	if err != nil {
		return fmt.Errorf("gravar o ofício %q: %w", nome, err)
	}
	return nil
}

// SaveProficiencies grava as categorias de proficiência, sem repetidas.
//
// # A recusa é UMA, e não uma lista para um 422
//
// A assinatura devolvia `(string, []string, error)`: o blob gravado, as
// categorias desconhecidas, e o erro. Os dois primeiros eram forma de resposta
// HTTP — o blob para o corpo do `PATCH`, a lista para o campo do 422 —, e
// aquele transporte morreu com a SPA. Hoje o único chamador é a tela, e o que
// ela faz com a lista é juntá-la numa frase (ALE-350).
//
// Uma categoria fora do catálogo não é erro do JOGADOR: a tela só oferece o que
// o catálogo tem. É o guarda contra a tela e a validação divergirem, e por isso
// a mensagem NOMEIA as desconhecidas — quem vai lê-la é quem mexeu no código.
func (p Plays) SaveProficiencies(ctx context.Context, characterID int64, categorias []string) error {
	var desconhecidas []string
	vistas := map[string]bool{}
	semRepetir := []string{}
	for _, cat := range categorias {
		if !book.IsProficiencyCategory(cat) {
			desconhecidas = append(desconhecidas, cat)
		}
		if !vistas[cat] {
			vistas[cat] = true
			semRepetir = append(semRepetir, cat)
		}
	}
	if len(desconhecidas) > 0 {
		return fmt.Errorf("proficiência fora do catálogo: %v", desconhecidas)
	}
	if err := p.queries.SetProficiencies(ctx, sqlcgen.SetProficienciesParams{
		Proficiencies: sheet.MarshalStrings(&semRepetir),
		UpdatedAt:     dbvalue.NowISO(),
		ID:            characterID,
	}); err != nil {
		return fmt.Errorf("gravar as proficiências da ficha %d: %w", characterID, err)
	}
	return nil
}
