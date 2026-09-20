package campaigns

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/app/boards"
	"t20engine/app/campaign"
	"t20engine/app/session"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA das campanhas, e a mais larga da série: a cena cobre quatro telas com
// um endereço cada — a lista, a campanha aberta, a folha em branco e a carta de
// entrar — e três delas ESCREVEM.
//
// Tipo com tag `json:` NÃO atravessa esta porta: a tag é a forma de um
// protocolo, e uma tela que a lesse passaria a depender do formato de um fio que
// ela não fala. Por isso a cena declara o `PlaceRow` dela, e por isso ela pede
// PERGUNTAS em vez de montar SQL.
//
// A lista das campanhas era o outro caso, e deixou de ser: ela virou caso de uso
// (`campaign.Directory`), e o tipo que a cena declarava para ela deixou de
// existir — quem chega por parâmetro já vem sem tag nenhuma (ALE-348).
type Deps interface {
	// Queries é o banco, e hoje esta cena só LÊ por ele — cinco consultas, zero
	// escritas. A última saiu com o `Delete` virando caso de uso, e com ela saiu
	// uma entrada que não existe mais — ela servia só para a cena orquestrar a
	// ordem de apagar, e essa ordem é do caso de uso (ALE-359).
	//
	// Ler não decide nada, e por isso o `Queries` fica: quem desenha precisa do
	// que o banco tem. Quem varre é o `convention.TestNoSceneWritesSql`.
	Queries() *sqlcgen.Queries
	CurrentUserID(r *http.Request) int64
	// RequesterIsAdmin olha a REQUISIÇÃO. O hospedeiro já tem um `IsAdmin`, que
	// olha a configuração e recebe um e-mail — dois nomes porque são duas
	// perguntas, e o compilador recusaria um só.
	RequesterIsAdmin(r *http.Request) bool
	CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error)

	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// PlaceRow é um lugar do acervo, na forma que esta tela desenha.
type PlaceRow struct {
	ID   int64
	Nome string
	// Pecas é a CONTAGEM, e ela separa a cena montada da cena abandonada:
	// "Cripta · 9 peças" é uma noite de trabalho, "cena vazia" é lixo.
	Pecas int
	// Quando é a última mudança, já legível.
	Quando string
	// NaMesaID é a sessão que mostra este lugar agora, ou zero. Ela decide os
	// gestos da linha: o lugar que está numa mesa não se monta nem se apaga. As
	// travas de verdade são do servidor; isto é a cortesia de não oferecer o
	// gesto que ele vai recusar.
	NaMesaID int64
}

// GroundOption é uma aparência de lugar, para o formulário do lugar novo.
type GroundOption struct {
	ID     string
	Rotulo string
}

// As montagens (`LoadList`, `LoadOne`, `LoadJoin`, `JoinBody`) são EXPORTADAS
// porque quem prova o caminho banco → tela é a bancada do `api`: este pacote não
// tem banco, e importar o `db/testdb` com um `*api.Server` seria o ciclo que a
// divisão existe para evitar.

// Scene é a cena montada com as dependências dela.
type Scene struct {
	deps Deps
	// access é a TRAVA de quem alcança o quê, e chega por parâmetro e não pela
	// porta: o `app/session` está ABAIXO desta cena, então ela o importa direto
	// e não há ciclo para desviar com uma interface. Mesmo desenho que a Mesa e
	// a ficha já têm.
	//
	// Ela entrou no lugar de um `RoleIn` da porta que MENTIA na assinatura: ele
	// declarava `(papel string, membros int, err error)` e o `int` era um status
	// HTTP, que a cena descartava com `_`. O nome `membros` ficou anos esperando
	// alguém usá-lo (ALE-348).
	access session.Access
	// acervo responde QUAIS campanhas esta pessoa vê, e com que papel. Ele
	// entrou no lugar de duas entradas da porta (`List` e `OwnerNames`) que o
	// adaptador cumpria traduzindo um DTO com tag `json:` — a forma de um fio
	// que esta tela não fala.
	acervo campaign.Directory
	// vida é o CICLO da campanha: abrir, renomear, cunhar convite, escolher as
	// regras opcionais. Ele AUTORIZA sozinho, e é isso que tirou da cena uma
	// segunda trava — que discordava da primeira e barrava o administrador
	// (ALE-348).
	vida campaign.Lifecycle
	// assentos senta alguém à mesa: as sete travas, a cópia do herói e o membro,
	// numa transação. As recusas dele são SENTINELAS que esta cena lê para
	// escolher a frase — ela podia lê-las porque o `app/` está abaixo dela, e
	// era isso que faltava quando elas moravam no hospedeiro (ALE-348).
	assentos campaign.Seating
	// lugares é o acervo de cenas guardadas da campanha, e ele chega INTEIRO.
	//
	// Eram quatro entradas da porta, e a razão escrita para elas era que "o
	// store é o vocabulário do domínio AO VIVO, e esta cena não é ao vivo". O
	// argumento caiu com a ALE-344: o `boards.Store` deixou de ser domínio e
	// virou CASO DE USO, e a Mesa já o recebe assim. Quatro perguntas que só
	// repassavam viraram uma dependência que diz o que é (ALE-348).
	lugares *boards.Store
}

func New(
	d Deps, trava session.Access, acervo campaign.Directory,
	vida campaign.Lifecycle, assentos campaign.Seating, lugares *boards.Store,
) Scene {
	return Scene{
		deps: d, access: trava, acervo: acervo,
		vida: vida, assentos: assentos, lugares: lugares,
	}
}
