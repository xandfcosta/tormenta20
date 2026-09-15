package sheetui

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA da FICHA. A cena é a maior do app, e a porta não: as sete abas leem o
// mesmo personagem e escrevem na mesma linha.
//
// O pacote se chama `sheetui` e não `sheet` porque `sheet` já é a FORMA do dado,
// e esta cena a lê em vinte arquivos — com o mesmo nome, cada um precisaria de
// um apelido no import.
type Deps interface {
	// Queries é o banco. As sete abas leem e escrevem a mesma linha de
	// personagem — é a concessão da forja e da administração, e o sinal de que
	// ela está no lugar é nenhum handler daqui montar SQL.
	Queries() *sqlcgen.Queries
	// Catalogs é o motor primado, para computar a ficha.
	Catalogs() *engine.Catalogs
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro.
	CurrentUserID(r *http.Request) int64
	// LoadCharacter monta o agregado a partir da linha do banco.
	LoadCharacter(ctx context.Context, row sqlcgen.Character) (sheet.CharacterDTO, error)
	// CharacterChanged avisa a MESA que esta ficha mexeu, e é chamada num lugar
	// só — o funil dos comandos. Passam trinta mutações por ali, e a linha
	// esquecida numa delas seria uma ficha que não atualiza só naquele gesto.
	CharacterChanged(characterID int64)
	// As ESCRITAS, uma por gesto: a cena decide QUANDO, o hospedeiro sabe COMO.
	SaveProficiencies(ctx context.Context, id int64, categorias []string) (string, []string, error)
	SaveNewCraft(ctx context.Context, id int64, nome string) error
	CastSpell(r *http.Request, dto sheet.CharacterDTO, magia string, aprimoramentos []sheet.AugmentPick) error
	ConsumeItem(r *http.Request, row sqlcgen.Character, itemID int64, pvRolado, pmRolado *int64) error
	ApplyClassLevel(r *http.Request, id int64, classe string, nivel int64) error
	ApplySpellBuffEffect(ctx context.Context, id int64, magia string, escopo *string) (sheet.EffectDTO, int, error)
	// PowerTempHpAmount lê o personagem do banco para saber o atributo-chave: a
	// CONTA é do `sheet`, a leitura é do hospedeiro.
	PowerTempHpAmount(r *http.Request, row sqlcgen.Character, atributo string) (int, bool)
	// Quem sabe o nome das colunas — e que esta tabela NÃO tem `updatedAt`,
	// então a gravação não toca carimbo — é o hospedeiro.
	SaveCustomItem(ctx context.Context, itemID int64, nome string, quantidade int64, espacos float64) error
	SaveEquipped(ctx context.Context, itemID int64, valor sql.NullString) error
	// SaveItemOverlays atravessa os valores do DOMÍNIO — a lista e o nome do
	// material —, não o JSON nem o `sql.NullString`.
	SaveItemOverlays(ctx context.Context, itemID int64, melhorias []string, material string) error
	// SaveChoices grava só as escolhas que MUDARAM: campo nulo é coluna que não
	// se toca. O NOME da coluna não atravessa — string de coluna saindo da cena
	// é SQL viajando com outra roupa.
	SaveChoices(ctx context.Context, id int64, escolhas ChoiceWrite) error
	// ApplyPowerTempHp aplica a reserva de PV temporários sob o vale-o-maior da
	// p256. A TRANSAÇÃO é do hospedeiro; a conta é do `sheet`
	// (`PlanPoolSupremacy`), e o que atravessa é QUANDO ela vale.
	ApplyPowerTempHp(ctx context.Context, id int64, powerID, escopo string, quanto int) error
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// ChoiceWrite são as cinco colunas de ESCOLHA da ficha, e nulo quer dizer "não
// mexa nesta". Dar um nome a elas aqui é o que tira o nome da COLUNA da cena.
type ChoiceWrite struct {
	ClassPowers          *string
	OriginChoices        *string
	ClassChoices         *string
	RaceAbilityChoices   *string
	RaceAttributeChoices *string
}

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
