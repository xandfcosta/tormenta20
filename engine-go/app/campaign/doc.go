// Package campaign são os CASOS DE USO de uma campanha: quem a vê, quem a
// abre, quem senta à mesa dela e o ELENCO de NPCs que o mestre prepara.
//
// # Por que `campaign` e não `campaigns`
//
// O padrão da casa é o plural, para o pacote de aplicação não colidir com o do
// domínio — foi assim que a ALE-344 escolheu `app/boards` ao lado de
// `domain/board`. Aqui o plural é PIOR, e a diferença foi contada antes de
// escolher:
//
//   - `app/campaigns` colide com `serve/web/campaigns`, a CENA, que sete
//     arquivos importam — mais o pior caso, que é a própria cena importando um
//     pacote com o nome dela e todo `campaigns.X` lá dentro ficando ambíguo
//     para quem lê;
//   - `app/campaign` colide com `domain/campaign` em DOIS arquivos
//     (`serve/api/campaigns.go` e `serve/web/campaigns/routes.go`), que passam
//     a apelidar um import.
//
// Dois contra sete, e o pior caso do plural não tem conserto por apelido.
//
// # O que é daqui e o que é do `domain/campaign`
//
// O `domain/campaign` é a REGRA pura — o que é um nome válido, quais regras
// opcionais existem. Aqui mora a ORQUESTRAÇÃO: autorizar, ler, decidir, gravar.
// A trava é a mesma do ciclo da sessão (`session.Access`), e não uma segunda:
// duas cópias de uma regra de autorização divergem em silêncio, e o sintoma é
// uma superfície deixando entrar quem a outra barra.
//
// O `Cast` chegou provando isso: a tabela `campaign_creatures` tinha DOIS donos
// — a cena da Mesa e o `app/initiative` —, e a trava de campanha estava escrita
// nos dois, com um comentário na cena avisando contra exatamente isso. Quem
// varre hoje é o `TestEveryCastGestureGoesThroughTheCampaignLock`, e ele se
// varre por REFLEXÃO: todo método que recebe um `app.Caller` é exercitado com
// um intruso, então um gesto novo não nasce sem trava (ALE-353).
package campaign
