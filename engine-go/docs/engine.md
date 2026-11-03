# O motor de regras

O ECS do motor, os modificadores, o dano, o deslocamento, o golpe e a manobra. Faz parte do [guia do engine](../CLAUDE.md).

## A regra de EMPILHAMENTO mora onde o `targetKey` mora

O `TestNoEntityStacksWithItself` (`domain/engine/modifier_stacking_rules_test.go`)
prende a p226 — uma entidade não declara dois modificadores não-condicionais no
mesmo alvo e tipo — com o balde do `targetKey` real, lido no `Modifier` real
(passando pelo `UnmarshalJSON`). **Guarda que precisa de regra privada do motor
mora no motor**: `switch` espelhado noutro pacote diverge em silêncio.

## A MANOBRA é o primeiro teste OPOSTO do motor

O resto do combate mede um total contra um número parado; na manobra (p234) os
DOIS lados rolam, e o `ResolveManeuver` recebe dois `d20`. Duas regras da
página:

1. **a MARGEM viaja**, porque é regra — cinco ou mais dão efeito extra ao
   derrubar e ao desarmar, e o empurrar ganha 1,5m por cada 5;
2. **o defensor sempre rola LUTA**, mesmo com arma de disparo: a proibição de
   ataque à distância é de quem ATACA.

**A manobra divide o provisório com o golpe**: ela É ataque corpo a corpo, e
dois provisórios seriam duas verdades sobre a cena. `proposesAttack` e
`proposesManeuver` delegam ao mesmo `proposeStrike`: as conferências antes da
rolagem são as mesmas. **O BÔNUS conhece o LADO**: o `Desejo de Liberdade`
ajuda quem é agarrado, não quem agarra; o escopo está na chave do `targetKey`,
e um caso prende os dois lados.

## O GOLPE EM DOIS TEMPOS: o dano só existe depois da decisão do mestre

O crítico muda QUANTOS dados se rolam (p231), e quem diz se houve crítico é o
mestre ("O veredito é do mestre", no [CLAUDE.md da raiz](../../CLAUDE.md)). Rolado
junto do d20, o dano sairia multiplicado pela SUGESTÃO do livro, e quem rola na
mão não saberia quantos dados pegar.

- **O teste**: o `JudgeAttack` devolve a rolagem, a Defesa e a sugestão
  (`Hit`, `Critical`), sem dano. O provisório leva a margem e o multiplicador da
  ARMA (`CritRange`, `CritMult`), que o botão "Crítico ×N" escreve.
- **A decisão** (`Ruling`: `miss`, `hit`, `critical`) é do mestre, UMA vez
  (`live.RuleAttack`). No acerto e no crítico ela já guarda o que rolar
  (`DamageToRoll`, do `engine.DamageToRoll`), na ordem em que os dados se leem:
  a arma primeiro, os extras depois.
- **O dano** é de quem atacou, ou do mestre por ele (`live.AttackAwaitingDamage`):
  o `RollAttackDamage` troca a sugestão pela decisão (`applyTheRuling`) e rola.
  Na mão, o `DiceRolledByHand` entrega os dados na mesma ordem, confere cada um
  contra as faces e o caso de uso recusa os que sobram.
- **O ✓ exige decisão e dano** (`AttackToCommit`); a manobra não rola dano, e
  o ✓ dela é a decisão.
- **O erro ENCERRA o golpe**: cobra a ação padrão (`session.Store.RuleAttack`)
  e deixa o que o ✓ deixaria — PM do impulso, guarda da investida, gatilhos
  cumulativos (`afterTheStrike`). O provisório fica com a decisão, para o
  jogador ler "o mestre disse: errou", até a vez mudar (`dropTheSettledMiss`)
  ou o próximo ataque o substituir.
- **Rolar de novo vale antes da decisão** (p226): o `Reroll` recusa depois dela.
- **Os campos da mão são sinais NUMERADOS** (`damage_die_1`…), montados em Go
  (`handDamageSignal`) e declarados vazios a cada cartão
  (`handSignalsDeclaration`), para o dado de um ataque não vazar para o
  seguinte. Um campo vazio recusa o gesto: misturar o dado da mesa com o do
  servidor seria pior que pedir o que falta.

- **No desktop o cartão FLUTUA** e se arrasta pela cabeça (`lib/float-card.ts`,
  `data-float` e `data-float-handle`); no celular fica no fluxo e mostra SÓ A
  CABEÇA, e o corpo é o diálogo `attack-sheet` que ela abre. É o mesmo nó nas
  duas larguras: no desktop o diálogo é desenhado dentro do cartão
  (`display: contents`), para o conteúdo não existir duas vezes. A posição vive
  no `style`, e o cartão declara `data-preserve-attr="style"`: sem isso o
  remendo do servidor o devolve ao lugar padrão no primeiro toque do mestre. O
  `attack-card-float.spec.ts` prova com um remendo que muda o PRÓPRIO cartão —
  passar a vez não o remenda, e "ficou no lugar" mediria o nada.

A ordem e quem faz cada passo estão presas no `TestTheDamageIsRolledOnlyAfterTheGMRules`
e no `TestTheGMsMissEndsTheAttackWithoutDamage`; a regra de dano sobre a
decisão, no `attack_ruling_rules_test.go`. Os casos de regra do livro decidem
como ele sugere (`suggestedAttack`), porque é assim que os exemplos se escrevem.

## A MANOBRA ANUNCIA a condição e o desfecho ao mestre; ela não os aplica

Duas das cinco manobras deixam condição — *"você deixa o alvo CAÍDO"* e *"uma
criatura AGARRADA fica desprevenida e imóvel"* (p234). As outras três têm efeito
de ITEM ou de MOVIMENTO, e condição ali seria inventar regra.

**O resultado guardado não traz vencedor nem aplica nada:** o `ManeuverOutcome`
tem os dois totais, a `Margin` e o `ConditionOnAWin`, a condição que o livro
RESERVA à vitória. Quem marca condição é o mestre, pelo `toggleCondition` da
fila.

**A LEITURA do embate é só para o mestre** ("O veredito é do mestre, e só ele o
vê", no [CLAUDE.md da raiz](../../CLAUDE.md)). O `engine.ReadManeuver` aplica o
desempate da p234 — maior total; no empate, maior bônus; bônus iguais, outro
teste —, e a faixa do mestre diz sucesso, falha ou empate, e o efeito extra da
margem de 5. A faixa do jogador passa pelo `forPlayer`, que tira o veredito, a
Defesa e o lado de quem resiste.

- O `TestTheManeuverNeverDeclaresAWinner` varre os campos do `ManeuverOutcome`
  por reflexão: o vencedor não é guardado nem aplicado.
- O `TestNoManeuverTextTheTablePlayerSeesDeclaresAWinner` varre o TEXTO da
  faixa do JOGADOR atrás de palavra de desfecho, que compila.
- O `TestOnlyTheGMSeesTheAttackVerdict` desenha a mesma mesa para os dois
  papéis, que é o predicado.
- **O id não vai para a tela**: o `book.ConditionName` o traduz (regra do
  `GLOSSARY.md`: id em inglês no código, texto em português na tela).

## Todo ALVO de modificador é declarado, e o RÓTULO não conta como leitor

Cada alvo do `targetKey` está numa de duas listas do
`domain/engine/modifier_target_rules_test.go`: os que o motor APLICA, com o
arquivo que os lê, e os INERTES (`alvosDeclaradosInertes`), com o motivo. As
duas cobrem exatamente o `switch`, nos dois sentidos, e reprovam pelo nome.
**A lista de inertes só pode encolher.** **O rótulo do
`serve/web/sheetui/effects_labels.go` NÃO conta como leitor**: o defeito caçado
é o modificador que aparece na aba Efeitos sem entrar em conta nenhuma.

Três famílias de inércia, três consertos:

- **o mecanismo ESQUECEU** (havia conta e ela não olhava o modificador): uma
  linha, e o teste nasce vermelho;
- **o mecanismo NÃO EXISTE**: construir o gesto, ou declarar por escrito que
  não será construído, como o `tempMp` (o app não modela PM temporário);
- **o alvo não devia existir** (só lendo o livro): era outro alvo, já existente,
  com condição; reescreva a entrada do catálogo.

**Casar nome não é casar conceito**: o nome de um alvo num arquivo do motor pode
ser outra coisa com a mesma grafia.

## O motor é ECS, e a ENTIDADE muda de fase para fase

**REGRA NOVA NASCE EM ECS. É o default, e o critério é APRENDER** (decisão do
dono): pergunte *"quem é a ENTIDADE, que COMPONENTES ela carrega, e que SISTEMA
os lê?"*, e não "que campo esta struct ganha". Regra sem entidade (formatador,
consulta, tabela fechada que só se lê) não é ECS: ECS para o que COMPÕE, função
para o que calcula. **Quem escolhe ECS paga o guarda de denominador junto**, um
irmão do `TestEveryDerivationSystemWritesItsComponent` com a lista à mão.

O pipeline da ficha é coleta → resolução → derivação → decomposição; o ataque é
um mundo à parte. A entidade muda porque a pergunta muda:

| fase | entidade | pergunta |
|---|---|---|
| **coleta** (`collect_ecs.go`) | a FONTE | quem concede alguma coisa a este personagem? |
| **resolução** (`resolve_ecs.go`) | o TERMO | quanto vale `defense`, e por quê? |
| **derivação** (`derive_ecs.go`) | a FICHA | qual é o número final? |
| **ataque** (`attack_ecs.go`) | a PARCELA DE DANO | de que se soma este dano, e o que ficou de fora? |
| **situação especial** (`special_situations_ecs.go`) | a LINHA da Tabela 5-3 | o que a cena em volta faz com este ataque? |
| **redação do tabuleiro** (`board_ecs.go`) | a PEÇA | quem olha daqui enxerga isto? |
| **ocupação do plano** (`board_occupancy_ecs.go`) | a PEÇA | que quadrado está tomado, e quem já tem peça na mesa? |
| **terreno sob a peça** (`situations_ecs.go`) | a PEÇA | que terreno ela está pisando, e o que ele faz com o ataque? |
| decomposição (`breakdowns.go`) | — | como se explica o número na tela? |

### Os componentes são agrupados por PERGUNTA

Um componente junta o que se lê junto: `derivedMagic` carrega limite de PM, CD e
custo. Um campo por componente seria a struct de volta, com cerimônia.

### O ataque em sistemas, e a única ordem que a rede NÃO pegava

- **O `d20` entra por parâmetro e os dados por função** (`JudgeAttack`, `RollAttackDamage`), para
  cada caso do livro ser um teste com o número escrito à mão. O `attack.go` só
  tem o vocabulário da fronteira e a montagem do mundo.
- Os casos do livro prendem quase toda a ordem; que cada sistema escreveu o
  componente dele prende o `TestEveryAttackSystemWritesItsComponent`, com um
  CENÁRIO por sistema, porque alguns só agem numa metade do dado.
- **Suprimir é write-once, e a PRIMEIRA razão é a que fica**: um sistema que
  passe depois não troca o motivo que a mesa lê pelo dele.
- **A notação de dano só é lida no tempo do dano**: julgar o teste não a lê, e o
  `RollAttackDamage` e o `DamageToRoll` falham num `"1d"` do catálogo sempre.

### A ORDEM dos sistemas é a regra, em toda fase

O oráculo não cobre ordem (nenhuma fixture tem todas as fontes juntas). Guarda
por fase: **coleta**, `TestEveryCollectorLandsInTheDeclaredOrder`; **resolução**,
`TestAFactorWhoseConditionFailsDoesNotApply` (colher o fator antes de julgar a
condição, preso pela CONSEQUÊNCIA, que sobrevive a uma reescrita dos sistemas);
**derivação**, `TestTheLoadIsDerivedBeforeWhatReadsIt`; **redação do
tabuleiro**, os dois guardas de ordem de desenho.

**Acrescentar uma fonte NÃO faz o guarda da coleta reprovar**: ele só mede os
pares que o DADO dele põe na mesma ficha, então o `everyCollectorCharacter`
cresce junto com o `collectionSystems`. Controle: trocar duas linhas da ordem
tem de reprovar nomeando as duas espécies fora de lugar.

### O PODER é uma coisa; a CONCESSÃO dele é outra

O `transcript.ClassPowers` tem o **verbete** (sem `className`, com a regra: nome,
descrição, página, bloco de escolha) e a **concessão** (com `className`, o nível
ou a escolha que destrava, e um `powerUid` apontando para o verbete). **A regra
é escrita uma vez, no verbete**; copiá-la para cada classe que concede o poder
dá duas regras com o mesmo nome.

**Quem resolve são DOIS pontos, um por leitor**: o `domain/book`, para a tela, em
`classPowersResolved`; o MOTOR, na carga do catálogo, em
`resolveClassPowerPointers` (a concessão sem nome ou sem modificador próprio
herda os do verbete). Não compartilham código porque isso exigiria uma aresta
entre `domain/engine` e `domain/catalog`.
`TestTheAttributeIncreaseLandsOnTheChosenAttribute` prende que o ponteiro chega
à ficha. **Antes de mudar a forma de um recurso, ache os leitores pelo nome
dele** (`git grep '"class-powers"'`), e não pela memória.

### O verbete tem DOIS identificadores, e eles têm papéis opostos

- **`id`** é o SLUG, o PÚBLICO: `[a-z0-9-]`, único no recurso (`esquiva`,
  `bencao-do-mana`). A URL (`/<recurso>/<id>`), o `grep` e a fixture o usam.
  Muda quando o nome muda.
- **`uid`** é a IDENTIDADE (`pwr_k3m9x2ft4q`): o banco o guarda, e um catálogo
  aponta para outro por ele. **Nunca** muda, e é opaco porque um id derivável do
  nome volta a mudar quando alguém "corrigir" a derivação.

Cunhar é `go run ./cmd/mintuids --aplicar` (de `engine-go/`; sem a flag, só
relata), e ela só CRIA. O escopo (quais arquivos e caminhos têm verbete) mora só
no `cmd/mintuids` e **não se repete em Go**: os guardas de
`domain/catalog/uid_rules_test.go` (`TestEveryCatalogEntryCarriesAUid`,
`TestNoUidNamesTwoDifferentThings`, `TestNoUidEverChanges`) conferem o resultado
contra `testdata/uid_baseline.txt` sem saber o escopo. **Verbete NOVO entra na
base quando alguém a regerar; o que reprova é um uid EXISTENTE mudar** — o banco
perde a referência e a ficha deixa de achar o verbete, sem erro.

#### Endereço não carrega RÓTULO, e a coincidência esconde isso

- O `TestEveryCatalogIdIsASlug` varre todo objeto com `id`, sem linha de base; o
  ponto é namespacing e passa (`class.barbaro.impeto`).
- **Um renome de chave preservou o comportamento quando o diff do oráculo não
  tem NENHUMA linha de número.**
- **Escolha guardada como JSON dentro de uma string também é ponteiro** (o
  `"race":"Lefou"` do `secondaryRaceChoices`), e varredura de campo não a
  alcança: renomeou chave, procure ali também.

### O bônus CIRCUNSTANCIAL já tem vocabulário — não invente um

Antes de acrescentar forma ao `Modifier`, veja se não é `context` com nota (o
interruptor que a PESSOA liga), `against` com traço ("contra o quê") ou
`terrain` (o tipo de terreno). **Cada `against` é um interruptor**; o que se
liga JUNTO divide uma flag (`flagOn`), como a Marca da Presa e os acréscimos
dela. Quase sempre falta o dado, não o motor. Erros que um teste só de números
não vê:

- **Alvo que já É o escopo dispensa condição**: "+2 em testes de ataque para
  derrubar" (p125) é `maneuver:derrubar`, sem `condition`. Opt-in para uma
  circunstância já escolhida faz valer de menos; circunstância sem interruptor
  faz valer de mais.
- **"Resistência a X +N" é bônus em TESTE (`resistance`), não redução de dano**
  (p226).
- **A manobra é teste OPOSTO (p234), e o alvo diz o lado**: `maneuver:<nome>` é
  quem ATACA (o +2 do `Derrubar Aprimorado`); com `scope: "defense"`, quem
  resiste (o +5 do `Desejo de Liberdade`, p89). O escopo entra na CHAVE do
  efeito, ou os dois somam no mesmo balde.
- **Para escolher o alvo, pergunte "que OUTROS números saem desse mesmo
  teste?"**, não só que teste se rola: a manobra é teste de Luta, e um bônus
  defensivo em `expertise:Luta` sobe o ataque da arma. Recorte fino pede alvo
  próprio.
- **Número impresso SEM alvo não vira modificador**, e o motivo fica escrito:
  penalidade que o motor não aplica (o "Água no Feijão": um +5 valeria sempre
  por um –5 de fabricação raro) ou TESTE que o livro não define (o "Vendedor de
  Carcaças"). Guarda: `TestEveryBonusAnOriginUniquePowerPrintsBecomesAModifier`,
  lista de PERMITIDOS com o POR QUÊ de cada linha.

### O DANO TEM TRÊS PARCELAS, e só uma multiplica no crítico

A p231: *"Multiplica os DADOS de dano do ataque (incluindo quaisquer aumentos
por passos) pelo multiplicador da arma. Bônus numéricos de dano, **assim como
dados extras**, não são multiplicados."*

| parcela | onde mora | multiplica? |
|---|---|---|
| os dados da ARMA | `WeaponCard.Damage` ("1d8") | **sim** |
| os dados EXTRAS de um encanto | `WeaponCard.ExtraDamage`, com o TIPO (é ele que diz à resistência a fogo onde agir) | não |
| o bônus numérico | `WeaponCard.DamageBonus` | não |
| o que só vale no crítico | `WeaponCard.CriticalBonus` | não, e fora do crítico nem existe |

**Estas duas formas do `Modifier` não são parcela da pilha**: o
`harvestWeaponRiders` as tira do mundo ANTES do `deferConditionals`, senão o
`onCritical` (condição que **não se liga**) viraria interruptor na aba Efeitos e
o dado extra somaria como `amount` ZERO (`weapon_riders_rules_test.go`):

```
{"target": {"k": "damage", "damageType": "fogo"}, "dice": "1d6"}
{"target": {"k": "damage"}, "amount": 10, "condition": {"c": "onCritical"}}
```

**O que denuncia o erro aritmético é a CONTAGEM de dados, não o total**: crítico
x2 com arma flamejante é 2d8 + 1d6, e 2d8 + 2d6 também dá total plausível. Por
isso o dublê do `attack_rules_test.go` estoura quando a regra pede um dado a
mais do que o caso preparou.

### A escala por PATAMAR, e o `default` que engole o `per` desconhecido

`scale.per: patamar` multiplica pelo ORDINAL do patamar (1 iniciante, 2
veterano, 3 campeão, 4 lenda): o "Coração Heroico" (p91) é um modificador só,
`maxPm 3`, dando 3/6/9/12. O `levelStep` não serve: os limiares (5, 11, 17) não
são passo fixo. **O `switch` do `evalModifierScale` tem `attribute` no
`default`**: um `per` desconhecido lê `attrTotals[""]`, acha zero, e o
modificador some sem erro. Guarda: `TestNoCatalogScaleIsOneTheEngineCannotRead`.

Para achar um poder no PDF, case por `bookpdf.Key` **dentro da página que o
catálogo declara**: o nome vem em VERSALETE (caixa baixa no `pdftotext`), e o
capítulo tem a TABELA de pré-requisitos antes dos verbetes.

### A SOMA DE ATRIBUTO: o `scale` fora dos vitais

"Você soma seu X em Y" é `scale: {per: attribute, attribute}` num alvo qualquer;
o `cap: classLevel` o limita ao nível NA CLASSE que concede o poder (p226), só
onde o verbete escreve "limitado pelo seu nível" (a Presença Paralisante e a
Mente Criminosa não são).

- **Ela se resolve numa SEGUNDA passada** (`sheetEffects`), porque o número é o
  atributo EFETIVO, com os itens. Por isso uma soma não mira atributo
  (`TestNoAttributeSumTargetsAnAttribute`), e toda decomposição da ficha lê o
  `sheetEffects`, não o `ComputeItemEffects`.
- **O atributo negativo SUBTRAI** (decisão do dono): o livro não tem regra
  própria e usa o valor como é — "10 + sua Destreza" (p106).
- **A trava que nomeia duas coisas usa `flags`**: o `flagOff` cala se QUALQUER
  uma estiver posta ("de armadura pesada ou na condição imóvel", p47); o
  `flagRequired` pede todas. A condição Imóvel (e quem a contém) levanta a flag
  `imovel`.
- **O dano por TIPO de arma** cai nos escopos `melee-light-or-agile` e `ranged`
  (arremesso e disparo, p142), lidos pela carta da arma junto do `DamageOwn`. O
  ATAQUE por escopo de arma (o `weapon:<id>` do Foco em Arma) é o `AttackOwn`,
  que a decomposição da tela soma.
- **O outro uso da arma mora dentro da carta dela** (`WeaponCard.OtherUse`,
  `otherUseOf`), e o índice da carta é o da mão: toda arma corpo a corpo se arremessa,
  com −5 se não tiver alcance (p143) nem o encanto Arremesso (p335, decisão do dono), e
  a Azagaia bate com −5 (p146). O pedido leva
  `OtherUse`, e o `AttackReplay` o guarda para a decisão e o dano. O impulso só corpo a
  corpo é recusado no uso à distância (`RefusesBoost`), para não cobrar PM em troca de nada.
- **O que é da ARMA fica nas cartas dela**: a melhoria e a falta de proficiência
  moram em `attack:weapon:<id>`, e não na perícia (`mirrorWeaponAttackMods`). Luta é
  uma só para a mão inteira, e um bônus nela sobe a arma da outra mão.

### A devoção é fonte da coleta, e não só de PV e PM

O poder concedido do devoto é FONTE da coleta (`godPowerActiveItems`) e chega a
qualquer alvo; o `vitalGrantMods` continua somando PV e PM, sem duplicar.

### Os acumuladores moram numa entidade SINGLETON

Flags e fatores são do mundo inteiro e ficam numa entidade só (`worldFlags`,
`worldFactors`), o idioma de "recurso" do ECS. Sem ela, um sistema escreveria
numa variável capturada e deixaria de ser função do mundo, que é o contrato do
`ecs.System`.

## O deslocamento é contado em QUADRADOS, e metro é coisa de tela

Todo deslocamento do livro é múltiplo de 1,5m: em metros, arredondaria na
fronteira do JSON e inventaria bônus; em quadrados, fecha exato. É a unidade do
tabuleiro (*"metro é coisa de tela"*, `board_movement.go`, p236), e a metade do
Lento (p395, *"arredonde para baixo para o primeiro incremento de 1,5m"*) vira
divisão inteira.

### A conversão é SIMÉTRICA, e isso não é enfeite

`amountInEngineUnits` na leitura, `amountInBookUnits` na escrita
(`Modifier.MarshalJSON`); sem a volta, reler o que o motor escreveu converte
duas vezes. Quem pega a assimetria é o `roundTrip` dos casos de paridade.
**O FIO fala a unidade do LIVRO**: o oráculo mostra −3 e +1,5 como a página
imprime, e o diff se revisa contra o livro.

### O modificador que MULTIPLICA

O `engine.Ratio` é um fator racional em inteiros (`{1,2}` metade, `{0,1}` zero)
— o Lento reduz o deslocamento à metade (p395), o Imóvel a 0m (p394), a Alta
Arcana corta o custo em PM pela metade —, num campo PRÓPRIO do `Modifier`: quem
tem fator não é parcela da pilha, o `Amount` fica zero e o valor vai do
`byTarget` para o `ItemEffects.Factors`. Racional porque o motor é em inteiros,
e a divisão inteira em quadrados é o arredondamento que o Lento pede.

**São DUAS FAMÍLIAS, distinguidas pelo ALVO:**

- **Fator de ALVO** (`{k: displacement}`, `{k: pmCost, scope: arcana}`): age
  DEPOIS da soma, sobre a BASE junto ("APÓS aplicar aprimoramentos e outros
  modificadores", Alta Arcana).
- **Fator de FONTE** (`{k: source, name: <id do poder>}`): age na PARCELA da
  fonte, ANTES do empilhamento. A Fúria Titânica (p42) dobra "o bônus que você
  recebe quando usa Fúria": 5 PM gastos dão +10, não o ataque inteiro dobrado;
  com o degrau pago, base e degrau do balde `morale` dobram e o não-empilhamento
  escolhe o maior.

**O fator de fonte tem DOIS sítios de aplicação**: o
`scaleWhatTheSourceFactorHits` no caminho incondicional, no ECS, e o
`ApplyActiveConditionals` na parcela DEFERIDA, que o ECS não multiplica —
dobrar nos dois dobraria duas vezes. A Fúria é condicional: vai pelo segundo.

**Dois fatores no mesmo alvo: vale o MAIS SEVERO, e eles não compõem**, com ou
sem condição — "aplique apenas o mais severo" (p394), *"reduções no custo de PM
não são cumulativas"* (p226); Lento mais Imóvel dá 0m. **O Foco em Magia é a
exceção ESCRITA**: −1 PM na magia escolhida, *"cumulativo com outras reduções de
custo"* (p131). Não é modificador `pmCost` (entraria na disputa da p226): o
`SpellPmCostFor` o soma por fora, antes do fator e do piso.

**O fator pode ter ESCOPO**: a Alta Arcana corta "suas magias arcanas" (p39),
pela chave `pmCost:arcana` (mesma divisa do `displacement:armor`), e o clérigo
do multiclasse paga cheio a divina. Quem diz a lista é a `sheet.SpellTradition`,
porque **a tradição é da CLASSE, não da magia**: a Luz é arcana no arcanista e
divina no clérigo.

> **A ISENÇÃO não é terceira forma do fator — é FLAG.** O anão "não tem o
> deslocamento reduzido por uso de armadura ou excesso de carga" (p20): o
> `displacementBreakdown` lê `DisplacementIgnoresArmor` e
> `DisplacementIgnoresLoad`, e pula a chave `displacement:armor` e a penalidade
> de carga sem tocar em fator (a magia de lentidão vale para o anão). **São duas
> flags porque o livro as vende separadas** (o Fanático, p128, compra só a da
> armadura). Isenção de terreno difícil (Acrobático, p129) é do `PathCost` do
> tabuleiro. **Confira na PÁGINA, e não na descrição do catálogo, antes de
> concluir que um poder pede isenção nova**: o corte pode ser da CIRCUNSTÂNCIA,
> que o app não modela.

### Onde a unidade aparece, e onde ela não aparece

| lugar | unidade |
|---|---|
| catálogo (os literais de `domain/catalog/transcript`) | metros |
| `Modifier.Amount` em memória | quadrados |
| `Modifier` no fio (JSON) | metros |
| `ComputedSheet.Displacement` (`json:"displacementSquares"`) | quadrados — e o nome do campo diz |
| `ItemEffects.ByTarget["displacement"]` | quadrados, e o nome do alvo é a única pista |
| a tela (crachá da mochila, régua da Mesa) | metros |

**Trocar a unidade de um campo do fio troca o NOME dele**
(`displacementSquares`): calado, todo leitor antigo leria 6 onde lia 9, sem erro.

