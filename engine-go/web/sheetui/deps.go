package sheetui

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/sheet"
	"t20engine/web/ui"
)

// A PORTA da FICHA (ALE-278), a última cena grande a sair antes da Mesa.
//
// Ela é a maior de todas — 36 arquivos, ~10.000 linhas, sete abas — e por isso
// vale dizer o que a porta NÃO reflete: o tamanho da cena não virou tamanho da
// interface. As sete abas leem o mesmo personagem e escrevem na mesma linha, e
// é por isso que a porta cabe em treze métodos enquanto a cena tem quinze
// arquivos de comando.
//
// # Por que o pacote se chama `sheetui`
//
// `sheet` já é a FORMA do dado — `CharacterDTO`, `Load`, `Compute` — e esta cena
// o lê 148 vezes em 20 arquivos. Com o mesmo nome, cada um desses vinte
// arquivos precisaria de um apelido no import. O sufixo resolve de graça, e o
// GLOSSARIO registra a medição que decidiu.
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
	// CharacterChanged avisa a MESA que esta ficha mexeu.
	//
	// Ela é o gateway do barramento de eventos, e a cena a chama num lugar só —
	// o funil dos comandos. Passam mais de trinta mutações por ali, e a linha
	// esquecida numa delas seria uma ficha que não atualiza só naquele gesto: é
	// a lição do gancho que nascia desligado, no `engine-go/CLAUDE.md`.
	CharacterChanged(characterID int64)
	// As ESCRITAS que a cena pede, uma por gesto que o hospedeiro sabe fazer.
	//
	// Todas gravam, e é por isso que nenhuma delas está aqui por conveniência: a
	// cena decide QUANDO, o hospedeiro sabe COMO.
	SaveProficiencies(ctx context.Context, id int64, categorias []string) (string, []string, error)
	SaveNewCraft(ctx context.Context, id int64, nome string) error
	CastSpell(r *http.Request, dto sheet.CharacterDTO, magia string, aprimoramentos []sheet.AugmentPick) error
	ConsumeItem(r *http.Request, row sqlcgen.Character, itemID int64, pvRolado, pmRolado *int64) error
	ApplyClassLevel(r *http.Request, id int64, classe string, nivel int64) error
	ApplySpellBuffEffect(ctx context.Context, id int64, magia string, escopo *string) (sheet.EffectDTO, int, error)
	// PowerTempHpAmount é quanto de PV temporário um poder concede.
	//
	// Ela mora no hospedeiro e não no `sheet` porque lê o personagem do banco
	// para saber o atributo-chave — a CONTA é do `sheet`, a leitura é dele.
	PowerTempHpAmount(r *http.Request, row sqlcgen.Character, atributo string) (int, bool)
	// SaveCustomItem e SaveEquipped substituem o SQL que a cena montava à mão.
	//
	// A cena compunha `setBuilder` + `"UPDATE character_items"`, e cena que
	// compõe SQL é cena com o banco dentro. Quem sabe o nome das colunas — e que
	// esta tabela NÃO tem `updatedAt`, então a gravação não toca carimbo — é o
	// hospedeiro.
	SaveCustomItem(ctx context.Context, itemID int64, nome string, quantidade int64, espacos float64) error
	SaveEquipped(ctx context.Context, itemID int64, valor sql.NullString) error
	// SaveItemOverlays grava a melhoria e o material de UM item.
	//
	// Ela é o terceiro `UPDATE character_items` que a cena montava à mão, e
	// atravessa os valores do DOMÍNIO — a lista e o nome do material —, não o
	// JSON nem o `sql.NullString`: quem sabe que material vazio é NULL e que
	// esta tabela não tem carimbo é o hospedeiro.
	SaveItemOverlays(ctx context.Context, itemID int64, melhorias []string, material string) error
	// SaveChoices grava as escolhas que MUDARAM, e só elas.
	//
	// A cena compara a ficha antes e depois e manda o que mexeu; campo nulo é
	// coluna que não se toca. O nome da coluna não atravessa — a versão anterior
	// mandava a string `"raceAttributeChoices"` de dentro da cena, que é SQL
	// viajando com outra roupa.
	SaveChoices(ctx context.Context, id int64, escolhas ChoiceWrite) error
	// ApplyPowerTempHp aplica a reserva de PV temporários que um poder concede,
	// sob o vale-o-maior da p256.
	//
	// A cena montava a TRANSAÇÃO inteira aqui — `BeginTx`, o plano, as escritas
	// e o `Commit` —, e o hospedeiro tinha a mesma sequência escrita de novo no
	// `applyPool` da rota JSON, apagada na ALE-277. A conta continua sendo do
	// `sheet` (`PlanPoolSupremacy`); o que atravessa é QUANDO ela vale.
	ApplyPowerTempHp(ctx context.Context, id int64, powerID, escopo string, quanto int) error
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// ChoiceWrite são as cinco colunas de ESCOLHA da ficha, e nulo quer dizer "não
// mexa nesta".
//
// Ela é declarada pela CENA e mapeada pelo hospedeiro, como o `ListRow` das
// campanhas: as cinco são blobs de JSON que a ficha lê como texto, e dar a elas
// um nome aqui é o que tira o nome da COLUNA de dentro da cena.
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
