// Package character são os CASOS DE USO de um herói: nascer, e depois jogar.
//
// # Ele tem DOIS chamadores, e isso não é novidade — estava escrito na porta
//
// A `forge.Deps` já dizia, antes desta camada existir, sobre a entrada que
// criava o personagem: *"fica no hospedeiro porque criar personagem é caminho
// compartilhado: a forja é uma entrada dele, não a dona. Ela recebe CONTEXTO e
// não o `*http.Request`: pedir a requisição inteira obrigaria quem chama a ter
// uma, e o gerador da seed não tem."*
//
// A forja é uma entrada; a seed é a outra. O que faltava era o endereço.
//
// # Dois tipos, e a linha entre eles é o GESTO
//
// O `Births` faz nascer e mantém os poços de quem está sendo construído — a
// forja e a seed. O `Plays` é o que acontece com um herói que já existe e está
// numa mesa: conjurar, beber, subir de nível, ligar um efeito. Os dois moram
// no mesmo pacote porque dividem o AGREGADO — a linha de `characters`, os
// poços dela, as classes, os itens e os efeitos — e um deles chama o
// recomputar de poços do outro (`vitals.go`).
//
// # Por que `character` e não um `app/sheet`
//
// A ALE-344 escolheu `app/boards` no plural para não colidir com
// `domain/board`; o plural de `sheet` já é o nome da coisa, então o truque não
// serve. E a colisão seria real: MEDIDO em três arquivos que precisariam dos
// dois pacotes ao mesmo tempo — `serve/web/sheetui/deps.go`,
// `serve/api/seeder.go` e `serve/api/sheetui_scene_deps.go` —, todos os três
// obrigados a apelidar um import. O glossário fecha a escolha: *ficha* e
// *personagem* traduzem os DOIS para `character`.
//
// # E aqui a TRANSAÇÃO é do caso de uso
//
// Nascer é uma escrita só em cinco tabelas; beber uma dose é a linha do efeito,
// a baixa do item e os poços juntos. Metade gravada é uma ficha que abre
// quebrada, e o dono da transação é quem sabe onde o gesto começa e acaba. É o
// que a Mesa não pôde provar: lá o estado vive em memória e a gravação é
// assíncrona.
//
// # A POSSE não é conferida aqui, e isso é decisão
//
// Ao contrário do `app/session`, nenhum método do `Plays` recebe `app.Caller`.
// A cena da ficha confere o dono UMA vez, no funil por onde passam as mais de
// trinta mutações dela (`sheetCommand`), e entrega a LINHA já lida. Repetir a
// conferência em seis dos trinta gestos daria duas opiniões sobre quem pode —
// e as outras vinte e quatro continuariam com uma só. Quando um segundo
// transporte aparecer, é o `Caller` que desce para cá, não a metade.
package character
