package campaigns

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/db/sqlcgen"
	"t20engine/sheet"
	"t20engine/web/ui"
)

// A PORTA das CAMPANHAS (ALE-278), e a mais larga da série: onze métodos,
// contra treze da administração e dois do trilho do mestre.
//
// O tamanho não é vício e também não é virtude — é o que a cena É. Ela cobre
// QUATRO telas com um endereço cada (a lista, a campanha aberta, a folha em
// branco e a carta de entrar), e três delas ESCREVEM. Comparar com o trilho do
// mestre é o que explica: aquele desenha o livro embutido e não toca banco;
// esta é a tela onde uma campanha nasce, muda de nome, ganha membro e é
// apagada.
//
// # O que ela NÃO pede, e as duas foram medidas
//
// **O `ListRow` não atravessa.** A cena o consumia direto, e ele é a
// forma do JSON da API — os campos têm tag `json:` e o nome deles é o do fio.
// É letra por letra o que a administração recusou com o `backupDTO`: a tela
// passaria a depender do formato de uma resposta HTTP que ela não serve. A cena
// declara o `ListRow` abaixo, e o hospedeiro MAPEIA do `campaignList` que já
// existe — sem duplicar a consulta, que é o outro erro possível aqui.
//
// **O SQL não atravessa.** A cena montava `setBuilder` + `execTouched` +
// `"UPDATE campaigns"` à mão. Cena que compõe SQL é cena com o banco dentro, e
// o remédio é a PERGUNTA: `SaveText`. O `Queries` continua na porta porque três
// telas leem e escrevem as próprias tabelas — é a mesma concessão da forja e da
// administração, e o sinal de que ela está no lugar é nenhum handler daqui
// tocar banco fora dele.
type Deps interface {
	// Queries é o banco. Sete consultas desta cena passam por ele: a campanha,
	// a campanha por token de convite, os membros, as sessões, as sessões vivas,
	// criar e apagar.
	Queries() *sqlcgen.Queries
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro.
	CurrentUserID(r *http.Request) int64
	// RequesterIsAdmin diz se QUEM PEDE administra o servidor. A lista de um
	// admin traz as campanhas de todo mundo, e é por isso que o
	// `ListRow.OwnerName` existe.
	//
	// O nome não é `IsAdmin` porque o hospedeiro já tem um, e ele responde OUTRA
	// pergunta: `IsAdmin(email string)` olha a configuração, esta olha a
	// requisição. Dois métodos de mesmo nome e formas diferentes o compilador
	// recusa — e forçar um nome só juntaria duas perguntas que só se parecem.
	RequesterIsAdmin(r *http.Request) bool
	// List são as campanhas que esta pessoa VÊ, com o papel dela em cada uma.
	//
	// A pergunta é do jeito da cena e o tipo é dela; quem responde é o
	// hospedeiro, que já sabe juntar dono, papel e personagem numa consulta só.
	List(ctx context.Context, userID int64, admin bool) ([]ListRow, error)
	// RoleIn é o papel de quem pede numa campanha, e quantos membros ela tem.
	RoleIn(ctx context.Context, userID int64, c sqlcgen.Campaign) (papel string, membros int, err error)
	// OwnerNames traduz o dono de cada campanha em nome, para a lista do admin.
	OwnerNames(ctx context.Context, campanhas []sqlcgen.Campaign, quemPede int64) map[int64]string
	// CharacterList são os personagens de quem pede, para a carta de entrar
	// oferecer quem senta à mesa.
	CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error)
	// IgnoredRules e SaveIgnoredRules são as regras opcionais da campanha (o
	// que o mestre DESLIGOU). Elas gravam, então são do hospedeiro.
	IgnoredRules(ctx context.Context, campanhaID int64) []string
	SaveIgnoredRules(ctx context.Context, campanhaID int64, regras []string) error
	// SaveText grava o nome e a descrição, e substitui o SQL que a cena montava.
	//
	// Descrição VAZIA é NULL, e a tradução é do hospedeiro de propósito: a regra
	// (`campaign.Description`) devolve texto justamente para não carregar
	// `database/sql`.
	SaveText(ctx context.Context, campanhaID int64, nome, descricao string) error
	// Join senta alguém à mesa, e devolve o MOTIVO da recusa e não o erro.
	//
	// Quem classifica é o hospedeiro, quem escolhe a frase é a cena — a decisão
	// que a porta de entrar deixou escrita (ALE-278). Os sentinelas de erro são
	// valores do `api`; se a cena os lesse, alcançaria o hospedeiro.
	Join(ctx context.Context, campanhaID, heroiID, quemPede int64, convite string) JoinRefusal
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// ListRow é uma campanha na LISTA, na forma que esta cena precisa.
//
// Ela existe para o `ListRow` do `api` não atravessar a fronteira: ele
// é a resposta de `GET /campaigns`, com tag `json:` em cada campo, e uma tela
// que o lesse passaria a depender do formato de um endpoint que ela não serve.
type ListRow struct {
	ID          int64
	Name        string
	Description string
	// Role é o papel de quem pede: `gm` ou `player`.
	Role string
	// OwnerName vem preenchido SÓ numa campanha que quem pede não possui, o que
	// hoje quer dizer um admin vendo as de todo mundo. Ausente é o caso normal,
	// então a tela marca a exceção em vez de toda linha.
	OwnerName string
	// Character é o herói de quem pede NESTA campanha, quando há um.
	Character *RowCharacter
}

// RowCharacter é o herói de quem pede numa campanha da lista.
type RowCharacter struct {
	ID      int64
	Name    string
	Level   int64
	Classes []sheet.ClassDTO
}

// JoinRefusal é o MOTIVO de a pessoa não conseguir sentar à mesa, declarado
// pela CENA.
//
// O hospedeiro tem sete sentinelas de erro para as sete travas do `Deps.Join`;
// esta lista tem SEIS valores, e a diferença é deliberada: a cena colapsa
// "personagem não existe" e "personagem é de outra pessoa" numa recusa só,
// porque as duas viram a mesma frase — "Escolha um herói seu" — e distinguir
// diria a um estranho se um id existe.
//
// **Quem classifica é o hospedeiro, quem escolhe a frase é a cena.** Se a cena
// lesse os sentinelas, alcançaria o `api`; se o `api` devolvesse a frase
// pronta, a voz da tela passaria a morar nele.
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

// AS MONTAGENS SÃO EXPORTADAS, e o consumidor de hoje é uma BANCADA.
//
// `LoadList`, `LoadOne`, `LoadJoin` e `JoinBody` são chamadas por
// `api/campaigns_*_test.go`, que prende o caminho BANCO → TELA: campanhas
// gravadas de verdade saem na lista com o papel certo, a sessão viva aponta para
// a mesa certa, o convite morto não oferece o botão de entrar.
//
// Este pacote não pode provar isso — não tem banco, e importar o `db/testdb`
// junto com um `*api.Server` seria o ciclo que a divisão inteira existe para
// evitar. Então a fronteira fica assim: **a cena diz COMO montar a si mesma, o
// hospedeiro prova que o que está no banco chega até lá.** É a mesma direção do
// `master.LoadBestiaryFrom` e do `characters.Load`.
//
// O que NÃO foi exportado é o contraste que importa: o `cenaFixture` da bancada
// e as frases que ela afirma são escritos do lado do `api`, porque importar do
// que está sendo testado faz o teste andar junto com o defeito.

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
