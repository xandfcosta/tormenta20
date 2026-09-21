package sheetui

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/app/character"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA da FICHA. A cena é a maior do app, e a porta não: as sete abas leem o
// mesmo personagem e escrevem na mesma linha.
//
// **Ela ENCOLHEU de dezoito métodos para oito na ALE-347**, e o mecanismo é o
// que vale para a próxima cena: os gestos da ficha viraram `character.Plays` e
// SAÍRAM da porta, em vez de ganharem um adaptador novo. O `app/` está abaixo
// desta cena, então ela o importa direto e recebe o caso de uso por parâmetro —
// ver o `Scene` no fim deste arquivo.
//
// O que sobra aqui é o que só o HOSPEDEIRO sabe: o banco, o motor primado, quem
// está pedindo, o aviso à Mesa e a casca da página.
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
	// O TURNO DA MESA, para um gesto que sai da ficha (p233). São DUAS porque
	// quem precisa RECUSAR pergunta antes de aplicar, e cobra depois de o gesto
	// ter saído — cobrar antes tiraria a ação de alguém por uma magia que a
	// regra seguinte recusou.
	//
	// Elas entram na porta porque a ficha NÃO CONHECE a mesa e não deve
	// conhecer: quem sabe em que sessão este personagem está é o hospedeiro. A
	// cena diz o que o gesto custa; onde isso é cobrado é de quem cumpre a
	// porta. Fora de uma cena de ação nenhuma das duas cobra nem recusa.
	ActionFitsOnTurn(characterID int64, cost engine.ActionCost) error
	SpendActionOnTurn(characterID int64, cost engine.ActionCost) error
	// As ESCRITAS, uma por gesto: a cena decide QUANDO, o hospedeiro sabe COMO.
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Scene é a cena montada com as dependências dela.
type Scene struct {
	deps Deps
	// plays são os GESTOS da ficha, e chegam por parâmetro e não pela porta: o
	// `app/character` está ABAIXO desta cena, então ela o importa direto e não
	// há ciclo para desviar com uma interface. É o mesmo desenho que a forja
	// tem com o `character.Births`.
	plays character.Plays
}

func New(d Deps, gestos character.Plays) Scene { return Scene{deps: d, plays: gestos} }
