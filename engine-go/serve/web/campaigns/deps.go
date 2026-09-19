package campaigns

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

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
	// Queries é o banco. O `Queries` continua na porta porque três telas leem e
	// escrevem as próprias tabelas; o sinal de que ele está no lugar é nenhum
	// handler daqui tocar banco fora dele.
	Queries() *sqlcgen.Queries
	CurrentUserID(r *http.Request) int64
	// RequesterIsAdmin olha a REQUISIÇÃO. O hospedeiro já tem um `IsAdmin`, que
	// olha a configuração e recebe um e-mail — dois nomes porque são duas
	// perguntas, e o compilador recusaria um só.
	RequesterIsAdmin(r *http.Request) bool
	CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error)
	// Join devolve o MOTIVO da recusa, não o erro: quem classifica é o
	// hospedeiro, quem escolhe a frase é a cena. Ler os sentinelas de erro daqui
	// alcançaria o `api`.
	Join(ctx context.Context, campanhaID, heroiID, quemPede int64, convite string) JoinRefusal

	// O ACERVO DE LUGARES são três perguntas e não o `boards.Store` inteiro: o
	// store é o vocabulário do domínio AO VIVO, e esta cena não é ao vivo.
	Places(ctx context.Context, campanhaID int64) []PlaceRow
	// NewPlace cria o lugar vazio quando ele ainda não existe. Nome repetido
	// leva ÀQUELE lugar: o nome é a identidade dele dentro da campanha.
	NewPlace(ctx context.Context, campanhaID int64, nome, chao string) (int64, error)
	RemovePlace(ctx context.Context, campanhaID, lugarID int64) error
	// Grounds são as aparências que um lugar pode ter. Vêm pela porta porque são
	// do tabuleiro: uma cópia aqui ofereceria um chão que o servidor não conhece
	// no dia em que a sexta nascer.
	Grounds() []GroundOption

	// CampaignDeleted é chamada ANTES do `DeleteCampaign`: apagar leva as
	// sessões por cascata, e depois não há como perguntar quais eram. Sem ela o
	// tabuleiro de cada sessão fica no mapa em memória batendo na chave
	// estrangeira, e a mesa se declara suja para sempre.
	CampaignDeleted(ctx context.Context, campanhaID int64)

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

// ListRow é uma campanha na LISTA, na forma que esta cena precisa.
// JoinRefusal é o MOTIVO de a pessoa não conseguir sentar à mesa.
//
// São SEIS valores para as sete travas do hospedeiro, e a diferença é
// deliberada: "personagem não existe" e "personagem é de outra pessoa" viram a
// mesma frase, e distinguir diria a um estranho se um id existe.
type JoinRefusal int

const (
	// JoinOK é a pessoa sentada.
	JoinOK JoinRefusal = iota
	// JoinNoSuchCampaign: o número digitado não é de campanha nenhuma.
	JoinNoSuchCampaign
	// JoinNeedsInvite: a mesa é fechada e o convite não serve.
	JoinNeedsInvite
	// JoinNotYourHero cobre as DUAS travas de personagem do hospedeiro.
	JoinNotYourHero
	// JoinAlreadyHasHero: esta pessoa já tem um herói nesta mesa.
	JoinAlreadyHasHero
	// JoinHeroAlreadyThere: este herói já está nesta mesa.
	JoinHeroAlreadyThere
	// JoinFailed é qualquer outra coisa, e vira o aviso interno.
	JoinFailed
)

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
}

func New(d Deps, trava session.Access, acervo campaign.Directory, vida campaign.Lifecycle) Scene {
	return Scene{deps: d, access: trava, acervo: acervo, vida: vida}
}
