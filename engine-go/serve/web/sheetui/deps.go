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
// **Ela ENCOLHEU muito na ALE-347**, e o mecanismo é o que vale para a próxima
// cena: os gestos da ficha viraram `character.Plays` e SAÍRAM da porta, em vez
// de ganharem um adaptador novo. Aqui estava a CONTAGEM de antes e depois, e
// ela já tinha virado mentira sem ninguém mexer nesta linha — um método novo
// não passa por aqui. O número é do `grep`; o que fica escrito é o mecanismo. O `app/` está abaixo
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
	ActionFitsOnTurn(ctx context.Context, characterID int64, cost engine.ActionCost) error
	SpendActionOnTurn(ctx context.Context, characterID int64, cost engine.ActionCost) error
	// PublishSkillTest põe um teste rolado na MESA onde este personagem está
	// (p220-221), e entra na porta pela mesma razão das duas acima: a ficha sabe
	// QUAL perícia e QUAL foi a conta, e quem sabe em que sessão o personagem
	// está é o hospedeiro.
	//
	// FORA DE UMA SESSÃO ela não faz nada e não é erro: a ficha aberta sozinha
	// continua rolando o dado e mostrando o número a quem está olhando — o que
	// não existe é a mesa para onde publicar. Recusar aqui transformaria "você
	// não está numa sessão" num defeito do gesto.
	PublishSkillTest(ctx context.Context, characterID int64, skill string,
		test engine.SkillTest, byHand bool) error
	// ProposeStrikeOnTable leva o golpe — ou a MANOBRA — rolado na superfície
	// Ações para a mesa onde este personagem está (p233-234), e ele entra na
	// porta pela mesma razão do `PublishSkillTest`: a ficha sabe COM O QUE se
	// ataca e CONTRA QUEM, e quem sabe em que sessão o personagem está é o
	// hospedeiro.
	//
	// FORA DE UMA SESSÃO ela RECUSA, e aqui está a diferença para o teste de
	// perícia: um teste rolado sozinho ainda mostra um número a quem está
	// olhando, mas um ataque sem mesa não tem alvo, não tem vez e não tem onde
	// pousar o provisório. Nada para fazer em silêncio seria o botão não
	// funcionar.
	ProposeStrikeOnTable(ctx context.Context, characterID int64, strike ActionStrike) error
	// As ESCRITAS, uma por gesto: a cena decide QUANDO, o hospedeiro sabe COMO.
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
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

func New(d Deps, gestures character.Plays) Scene { return Scene{deps: d, plays: gestures} }

// ActionStrike é o gesto de combate que saiu da superfície Ações.
//
// Ele é um struct e não quatro parâmetros porque três dos campos são opcionais
// entre si — manobra OU golpe, e a arma só importa no segundo —, e uma
// assinatura de quatro posições faz o chamador contar vírgulas.
//
// O ATACANTE NÃO ESTÁ AQUI: quem ataca é a VEZ, e quem a resolve é o
// hospedeiro, contra o banco (p231). Um campo de atacante nesta estrutura seria
// o cliente escolhendo de quem é o turno.
type ActionStrike struct {
	// TargetEntryID é a linha da FILA mirada, e ela vem do sinal que a barra da
	// Mesa escreve.
	TargetEntryID string
	// Weapon é qual das empunhadas, por índice. Zero é a primeira.
	Weapon int
	// Maneuver é uma das cinco da p234 quando o gesto é manobra. Vazio é golpe.
	Maneuver string
}
