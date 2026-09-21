package api

import (
	"context"
	"t20engine/app/accounts"
	"t20engine/app/character"
	"t20engine/domain/account"
	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O QUE O GERADOR DA SEED PEDE DA CASA.
//
// O `cmd/seed` monta o conjunto de desenvolvimento — três contas, o elenco de
// teste, as crônicas — e despeja tudo em `seed.sql`. A promessa dele é a razão
// de ele existir: os hashes de bcrypt, os vitais computados pelo motor e o leque
// normalizado vêm **do mesmo código que o app roda**, nunca mantidos à mão.
//
// # Por que não pelas cenas
//
// A resposta óbvia seria apontá-lo para o roteador das cenas, e ela não serve:
// a forja cria personagem de NÍVEL 1 com kit inicial, e o elenco da seed tem
// heróis de nível 8 a 10 com atributos, itens e magias escolhidos — uma ficha
// inteira escrita de uma vez.
//
// Então o gerador é mais um CONSUMIDOR com porta declarada, como as cenas: ele
// diz o que precisa, e o hospedeiro cumpre. Cada método aqui é um invólucro
// sobre a regra que já existe — nenhuma linha de regra mora neste arquivo, e é
// isso que mantém a promessa acima de pé.
type Seeder struct {
	gate accounts.Gate
	// births é o MESMO caso de uso que a forja usa — e é por isso que este
	// gerador não monta uma requisição falsa para chamar a criação: ele é o
	// segundo chamador que a porta da forja nomeava antes de haver camada.
	births character.Births
	// plays é o mesmo caso de uso que a Mochila da ficha usa para beber uma
	// dose — o segundo chamador de novo, e pela mesma razão do `births`.
	plays   character.Plays
	sheet   sheetRules
	queries *sqlcgen.Queries
}

func (s *Server) Seeder() Seeder {
	return Seeder{
		gate: s.accountGate(), births: s.characterBirths(), plays: s.characterPlays(),
		sheet: s.sheetRules(), queries: s.queries,
	}
}

// CreateAccount faz nascer a conta, com o bcrypt da casa.
//
// O e-mail da seed entra em `ADMIN_EMAILS` para dispensar convite: a primeira
// conta de um banco vazio não tem quem a tivesse convidado, e o gerador é o
// próprio admin dele. Nada do papel chega ao `seed.sql` — ele é derivado do
// ambiente a cada requisição e não tem coluna.
func (sd Seeder) CreateAccount(ctx context.Context, email, name, password string) error {
	_, err := sd.gate.Register(ctx, account.RegisterBody{
		Email: email, Password: password, Name: &name,
	})
	return err
}

// CreateCharacter escreve a ficha INTEIRA, e o poço vem de graça.
//
// A ordem importa só para o nível total e as proficiências, que saem das CLASSES
// antes da escrita. Os vitais não entram: o poço é derivado do catálogo a cada
// leitura, e um personagem sem dano nasce cheio por construção (ALE-355).
//
// É por isso que o gerador manda 9999 nos quatro vitais: um valor que a cura só
// pode aparar para baixo. Barra danificada é escrita DEPOIS, pelo `SetHp`.
func (sd Seeder) CreateCharacter(
	ctx context.Context, ownerID int64, body sheet.CreateBody,
) (int64, error) {
	var totalLevel int64
	classes := make([]string, len(body.Classes))
	for i, c := range body.Classes {
		totalLevel += c.Level
		classes[i] = c.ClassName
	}
	id, err := sd.births.Create(ctx, ownerID, body.Name, body, totalLevel,
		book.GrantedProficiencies(classes), sheet.ToStringSet(body.TrainedExpertises))
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Character devolve a ficha carregada, para o gerador ler o PV máximo que o
// motor calculou e os itens que a criação materializou.
func (sd Seeder) Character(ctx context.Context, id int64) (sheet.CharacterDTO, error) {
	row, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return sheet.CharacterDTO{}, err
	}
	return sd.sheet.LoadCharacter(ctx, row)
}

// LearnSpell põe a magia no grimório da ficha, preparada ou não.
func (sd Seeder) LearnSpell(ctx context.Context, id int64, catalog string, prepared bool) error {
	preparedAt := int64(0)
	if prepared {
		preparedAt = 1
	}
	_, err := sd.queries.CreateSpell(ctx, sqlcgen.CreateSpellParams{
		Characterid: id, Catalogspellid: catalog,
		Prepared: preparedAt, Learnedat: dbvalue.NowISO(),
	})
	return err
}

// SetHp deixa a barra danificada, para o elenco de teste ter ficha machucada.
func (sd Seeder) SetHp(ctx context.Context, id, current int64) error {
	row, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return err
	}
	return sd.plays.SetHpTo(ctx, row, current)
}

// ConsumeItem gasta uma dose, para o elenco ter efeito de cena ligado.
func (sd Seeder) ConsumeItem(ctx context.Context, id, itemID int64) error {
	row, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return err
	}
	_, err = sd.plays.Consume(ctx, row, itemID, nil, nil)
	return err
}
