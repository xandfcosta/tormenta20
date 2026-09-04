package api

import (
	"context"
	"t20engine/account"
	"t20engine/book"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/sheet"
)

// O QUE O GERADOR DA SEED PEDE DA CASA (ALE-287).
//
// # Por que ele existe
//
// O `cmd/seed` monta o conjunto de desenvolvimento — três contas, o elenco de
// teste, as crônicas — e despeja tudo em `seed.sql`. A promessa escrita no
// cabeçalho dele é a razão de ele existir: os hashes de bcrypt, os vitais
// computados pelo motor e o leque normalizado vêm **do mesmo código que o app
// roda**, nunca mantidos à mão.
//
// Ele cumpria isso dirigindo a API JSON em processo. A ALE-277 apagou as sete
// rotas que ele usava, e o gerador parou de rodar — sem que nada acusasse, e a
// varredura de órfãs não o viu porque ele chama por CAMINHO EM STRING enquanto
// a busca era por símbolo. É a forma que uma varredura por identificador não
// alcança, e vale saber que ela existe.
//
// # Por que não pelas cenas
//
// A resposta óbvia seria repontá-lo para o roteador das cenas, e ela não serve:
// a forja cria personagem de NÍVEL 1 com kit inicial, e o elenco da seed tem
// heróis de nível 8 a 10 com atributos, itens e magias escolhidos. Essa
// capacidade só a rota JSON tinha, e apagá-la levou junto o único caminho que
// escrevia uma ficha inteira de uma vez.
//
// Então o gerador vira mais um CONSUMIDOR com porta declarada, como as onze
// cenas: ele diz o que precisa, e o hospedeiro cumpre. Cada método aqui é um
// invólucro sobre a regra que já existe — nenhuma linha de regra mora neste
// arquivo, e é isso que mantém a promessa do cabeçalho dele de pé.
type Seeder struct {
	accounts accountRules
	forge    forgeHost
	sheet    sheetRules
	queries  *sqlcgen.Queries
}

func (s *Server) Seeder() Seeder {
	return Seeder{
		accounts: s.accountRules(), forge: s.forgeHost(),
		sheet: s.sheetRules(), queries: s.queries,
	}
}

// CreateAccount faz nascer a conta, com o bcrypt da casa.
//
// O e-mail da seed entra em `ADMIN_EMAILS` para dispensar convite: a primeira
// conta de um banco vazio não tem quem a tivesse convidado, e o gerador é o
// próprio admin dele (ALE-120). Nada do papel chega ao `seed.sql` — ele é
// derivado do ambiente a cada requisição e não tem coluna.
func (sd Seeder) CreateAccount(ctx context.Context, email, nome, senha string) error {
	_, err := sd.accounts.createAccount(ctx, account.RegisterBody{
		Email: email, Password: senha, Name: &nome,
	})
	return err
}

// CreateCharacter escreve a ficha INTEIRA e cura os vitais pelo motor.
//
// A sequência é a que o manipulador apagado fazia, e a ordem importa: o nível
// total e as proficiências saem das CLASSES antes da escrita, e a cura vem
// depois — o `HealVitals` recomputa PV e PM máximos a partir da ficha já
// gravada, que é como o número da seed passa a ser o número que o motor daria.
//
// É por isso que o gerador manda 9999 nos quatro vitais: um valor que a cura só
// pode aparar para baixo. Barra danificada é escrita DEPOIS, pelo `SetHp`.
func (sd Seeder) CreateCharacter(
	ctx context.Context, donoID int64, corpo sheet.CreateBody,
) (int64, error) {
	var nivelTotal int64
	classes := make([]string, len(corpo.Classes))
	for i, c := range corpo.Classes {
		nivelTotal += c.Level
		classes[i] = c.ClassName
	}
	id, err := sd.forge.InsertCharacter(ctx, donoID, corpo.Name, corpo, nivelTotal,
		book.GrantedProficiencies(classes), sheet.ToStringSet(corpo.TrainedExpertises))
	if err != nil {
		return 0, err
	}
	linha, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return 0, err
	}
	dto, err := sd.sheet.LoadCharacter(ctx, linha)
	if err != nil {
		return 0, err
	}
	return id, sd.forge.HealVitals(ctx, id, &dto)
}

// Character devolve a ficha carregada, para o gerador ler o PV máximo que o
// motor calculou e os itens que a criação materializou.
func (sd Seeder) Character(ctx context.Context, id int64) (sheet.CharacterDTO, error) {
	linha, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return sheet.CharacterDTO{}, err
	}
	return sd.sheet.LoadCharacter(ctx, linha)
}

// LearnSpell põe a magia no grimório da ficha, preparada ou não.
func (sd Seeder) LearnSpell(ctx context.Context, id int64, catalogo string, preparada bool) error {
	preparadaEm := int64(0)
	if preparada {
		preparadaEm = 1
	}
	_, err := sd.queries.CreateSpell(ctx, sqlcgen.CreateSpellParams{
		Characterid: id, Catalogspellid: catalogo,
		Prepared: preparadaEm, Learnedat: plataforma.NowISO(),
	})
	return err
}

// SetHp deixa a barra danificada, para o elenco de teste ter ficha machucada.
func (sd Seeder) SetHp(ctx context.Context, id, atual int64) error {
	linha, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return err
	}
	return sd.queries.SetCharacterVitals(ctx, sqlcgen.SetCharacterVitalsParams{
		HpMax: linha.Hpmax, HpCurrent: atual,
		MpMax: linha.Mpmax, MpCurrent: linha.Mpcurrent,
		UpdatedAt: plataforma.NowISO(), ID: id,
	})
}

// soOErro descarta o `doseUsed`, que é a forma de FIO da resposta JSON.
func soOErro(_ doseUsed, err error) error { return err }

// ConsumeItem gasta uma dose, para o elenco ter efeito de cena ligado.
func (sd Seeder) ConsumeItem(ctx context.Context, id, itemID int64) error {
	linha, err := sd.queries.GetCharacter(ctx, id)
	if err != nil {
		return err
	}
	return soOErro(sd.sheet.consumeItemForCharacter(ctx, linha, itemID, nil, nil))
}
