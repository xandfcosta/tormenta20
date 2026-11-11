package live

import (
	"encoding/json"
)

// O ATAQUE PROPOSTO, e a divisa de quem decide sobre ele.
//
// A mesma do movimento no tabuleiro, e pela mesma razão: o que um jogador faz é
// uma PROPOSTA que a mesa vê, e quem transforma proposta em consequência é o
// mestre. Aqui a consequência é PV saindo de uma linha, que é a coisa menos
// reversível da sessão.
//
// # Por que o provisório mora no REGIME e não no tabuleiro
//
// O ataque é entre LINHAS DA FILA, não entre peças: uma mesa sem mapa aberto
// continua tendo combate, e o alvo de um ataque é um combatente, não um
// quadrado. Guardá-lo no `BoardState` tornaria o combate refém de ter um
// tabuleiro aberto — e o tabuleiro é uma superfície, não a cena.
//
// # Por que os NÚMEROS chegam prontos
//
// Quem resolve `d20` contra Defesa é o `engine.JudgeAttack`, e este contexto
// não o alcança (ver o `boundary_test.go`): o regime fala de quem está na mesa
// e de quanto PV cada um tem, e a regra do livro fala de dados. A tradução é da
// camada de caso de uso, que conhece os dois — é o mesmo desenho da porta de
// vitais da ficha.

// PendingAttack é um ataque rolado e ainda não confirmado.
//
// Ele carrega a CONTA INTEIRA e não só o dano, porque é isso que a mesa lê
// enquanto decide: "19 contra Defesa 17, acertou; 1d8 deu 8, mais 3; a RD comeu
// 5" é uma frase que dispensa perguntar ao servidor.
type PendingAttack struct {
	AttackerEntryID string `json:"attackerEntryId"`
	TargetEntryID   string `json:"targetEntryId"`
	// TargetTokenID é a PEÇA DE OBJETO atacada (p239), e é a alternativa ao
	// `TargetEntryID`: um objeto não tem linha na fila porque não tem turno.
	// Exatamente um dos dois vem preenchido.
	//
	// É por ele que a CONFIRMAÇÃO sabe onde o dano pousa — na fila, pelo
	// `DeltaVitals`, ou no tabuleiro, pelo `DamageObject`. Os dois agregados têm
	// travas próprias, então quem escolhe é a cena, acima dos dois.
	TargetTokenID string `json:"targetTokenId,omitempty"`
	// TargetLabel é o nome do que foi atacado, e ele viaja porque a faixa o lê.
	//
	// A criatura tem a linha da fila de onde tirá-lo; o objeto não tem nada — o
	// provisório é a única coisa que a faixa recebe, e sem o nome ela diria
	// "acertou" sem dizer o quê.
	TargetLabel string `json:"targetLabel,omitempty"`
	Weapon      string `json:"weapon"`
	Roll        int    `json:"roll"`
	Total       int    `json:"total"`
	// Defense é a Defesa que o ataque enfrentou. Ela viaja porque a mesa lê a
	// COMPARAÇÃO — "24 vs Defesa (17)" é o que explica o veredicto, e sem o 17 a faixa
	// afirma um acerto sem dizer contra o quê.
	Defense int `json:"defense"`
	// NonLethal é quanto do `Damage` não conta para sangrar nem para morrer
	// (p236) — tudo, quando a arma é Piedosa (p336). Viaja no provisório porque
	// quem aplica é a CONFIRMAÇÃO, e entre rolar e confirmar o ataque pode ser
	// cancelado.
	NonLethal int `json:"nonLethal,omitempty"`
	// BoostPowerID e BoostPM são o impulso de PM que o golpe levou (o Golpe
	// Divino, o Ataque Especial — ALE-487) e o que ele custa. A proposta paga ao
	// nascer (`Paid`).
	BoostPowerID string `json:"boostPowerId,omitempty"`
	BoostPM      int    `json:"boostPm,omitempty"`
	// BoostHitPoints é o PV que o gasto tira de quem ataca, já rolado: o 1d8 da Tumular (p336). A proposta
	// o cobra junto do PM, e o cancelamento devolve (decisão do dono).
	BoostHitPoints int `json:"boostHitPoints,omitempty"`
	// BoostAgainst é o tipo de criatura contra o qual o gasto vale (a Anticriatura, p335) e o que a mesa
	// sabia do alvo no instante do d20. A faixa do MESTRE diz se o alvo é desse tipo; quem decide é ele.
	BoostAgainst *ChosenKindRead `json:"boostAgainst,omitempty"`
	// Charge é a investida que levou ao golpe (p235, p42): a proposta paga a ação
	// COMPLETA dela, e o encerramento abre a guarda de quem investiu (ALE-499).
	Charge string `json:"charge,omitempty"`
	// OpensExtra diz que este golpe é a ação agredir: a resposta do mestre abre o ataque extra
	// das armas naturais (p25). NaturalSource é a fonte da arma, quando ela é natural. ExtraOf é
	// a fonte quando este golpe É o ataque extra, e ExtraLimit, quantos ela dá na vez. O PM dele
	// vai no `BoostPM`, somado ao do impulso.
	OpensExtra    bool   `json:"opensExtra,omitempty"`
	NaturalSource string `json:"naturalSource,omitempty"`
	ExtraOf       string `json:"extraOf,omitempty"`
	ExtraLimit    int    `json:"extraLimit,omitempty"`
	// ExtraLimb e ExtraLimbLimit são a pata deste extra e quantos ela dá na vez (p137), quando há
	// pata armada — ver `engine.NaturalStrike.Limb`.
	ExtraLimb      string `json:"extraLimb,omitempty"`
	ExtraLimbLimit int    `json:"extraLimbLimit,omitempty"`
	// OpensStyle é o estilo ligado neste golpe de agredir, que a resposta do mestre abre (p125), e
	// StyleItem é a linha da mochila da arma, no golpe que abre e no adicional. FullAction diz que
	// ele pagou a ação completa do Disparo Rápido. ExtraIsStyle diz que este golpe É o adicional,
	// e ExtraOtherWeapon, que ele pede a outra arma.
	OpensStyle       string `json:"opensStyle,omitempty"`
	StyleItem        int64  `json:"styleItem,omitempty"`
	FullAction       bool   `json:"fullAction,omitempty"`
	ExtraIsStyle     bool   `json:"extraIsStyle,omitempty"`
	ExtraOtherWeapon bool   `json:"extraOtherWeapon,omitempty"`
	// Paid é o que a proposta pagou ao nascer, e é o que o cancelamento do mestre
	// devolve. Nulo na proposta de antes desta regra, que não pagou nada.
	Paid *AttackPaid `json:"paid,omitempty"`
	// AttackerStances e TargetStances são as posturas de cada lado na hora da proposta.
	// Os gatilhos cumulativos do encerramento leem daqui (decisão do dono), porque ligar
	// a Fúria depois do crítico não muda o golpe que foi proposto.
	AttackerStances []string `json:"attackerStances,omitempty"`
	TargetStances   []string `json:"targetStances,omitempty"`
	// Dose é o que a Injeção alquímica da arma libera se o golpe acertar (p165).
	// Nula em arma sem dose e na manobra, que não causa dano.
	Dose *AttackDose `json:"dose,omitempty"`
	// Situations são os rótulos da Tabela 5-3 que valeram neste ataque, e a
	// faixa os escreve. Sem eles a mesa lê uma Defesa que a ficha não tem, ou
	// um ataque que bateu a Defesa e errou (p238), e vai procurar o defeito.
	Situations []string `json:"situations,omitempty"`
	// Hit e Critical são a SUGESTÃO do livro enquanto o mestre não decide, e a
	// decisão dele depois (`Ruling`): é a mesma pergunta — "acertou? critou?" —
	// em dois momentos, e quem lê depois da decisão lê a palavra do mestre.
	Hit      bool `json:"hit"`
	Critical bool `json:"critical"`
	// CritRange e CritMult são a margem de ameaça e o multiplicador da ARMA
	// usada (p230): o botão do mestre diz "Crítico ×3" e a sugestão explica "20
	// natural, na margem 20" com eles.
	CritRange int `json:"critRange,omitempty"`
	CritMult  int `json:"critMult,omitempty"`
	// Ruling é a decisão do mestre: vazio enquanto ele não decide, e então
	// `RulingMiss`, `RulingHit` ou `RulingCritical`. Só o golpe tem decisão —
	// a manobra não rola dano, e o ✓ dela já é a decisão.
	Ruling string `json:"ruling,omitempty"`
	// DamageToRoll é o que o jogador rola depois de um acerto ou crítico, já
	// com o ×N da arma no crítico (p231). Nulo antes da decisão e no erro.
	DamageToRoll *DamageToRoll `json:"damageToRoll,omitempty"`
	// DamageRolled diz que o dano foi rolado: só então o ✓ do mestre pousa.
	DamageRolled bool `json:"damageRolled,omitempty"`
	// ExtraDice são as parcelas de dado de outro tipo — o 1d6 de fogo do
	// encanto —, discriminadas para a mesa ler a notação inteira.
	ExtraDice []RolledGroup `json:"extraDice,omitempty"`
	// Melee diz se o golpe foi CORPO A CORPO, e ele viaja porque há regra que
	// depende disso DEPOIS da confirmação: o Golpe Mágico do bardo só acumula
	// *"ao acertar um ataque corpo a corpo"* (p45). A arma não vem junto no
	// provisório — só o nome dela —, e um nome não responde a pergunta.
	Melee     bool  `json:"melee,omitempty"`
	Dice      []int `json:"dice,omitempty"`
	Faces     int   `json:"faces,omitempty"`
	RawDamage int   `json:"rawDamage"`
	// CriticalExtra é quanto do dano veio do crítico, e
	// CritExtraIgnoreChance é a chance do ALVO de ignorar justamente essa
	// parte (p166, p338). Os dois viajam juntos porque é o PAR que serve à
	// mesa: a chance sozinha não diz o que tirar, e o extra sozinho não diz
	// que há algo a rolar. Quem rola é o mestre — o app informa (ALE-436).
	// Announces são as condições que o LIVRO reserva a este golpe (p335-336).
	// Elas chegam à faixa para o mestre ler, e nenhuma delas foi aplicada —
	// aplicar é o gesto dele, na fila (ALE-441).
	Announces             []ForeseenCondition `json:"announces,omitempty"`
	CriticalExtra         int                 `json:"criticalExtra,omitempty"`
	CritExtraIgnoreChance int                 `json:"critExtraIgnoreChance,omitempty"`
	Absorbed              int                 `json:"absorbed"`
	Damage                int                 `json:"damage"`
	// Maneuver é a conta da MANOBRA (p234), e ela é nula num golpe comum.
	//
	// Uma manobra É um ataque corpo a corpo — o livro abre a página dizendo isso
	// —, e por isso ela mora no MESMO provisório em vez de num irmão: a divisa de
	// quem propõe e quem confirma é a mesma, e os verbos da faixa também. Dois
	// provisórios simultâneos seriam duas verdades sobre a mesma cena, que é o
	// que o `ProposeAttack` já existe para não permitir.
	//
	// Num golpe, os campos de dano falam e este é nulo; numa manobra, o inverso.
	// É ele que diz à faixa qual das duas frases escrever.
	Maneuver *ManeuverRoll `json:"maneuver,omitempty"`
	// ByUserID é quem rolou. O mestre confirma e cancela por qualquer um; quem propôs
	// só rola o dano e fecha o próprio erro.
	ByUserID int64 `json:"byUserId"`
	// Rerolls são os "rolar de novo" pagos, em ordem: vale o ÚLTIMO dado
	// (p226, ALE-508).
	Rerolls []DieReroll `json:"rerolls,omitempty"`
	// WeaponID é o id do catálogo da arma, que é o que o Mestre em Arma escolheu (p66).
	WeaponID string `json:"weaponId,omitempty"`
	// Frozen é o golpe como foi proposto (`engine.FreezeStrike`). O regime só o guarda.
	Frozen json.RawMessage `json:"frozen,omitempty"`
}

// AttackPaid é o que saiu da mesa e da ficha quando a proposta nasceu.
//
// As três ações guardam o que a cobrança de fato tirou: a pessoa pode ter andado depois
// de propor, e devolver o turno inteiro daria de volta esse movimento. A cena e a rodada
// dizem de que turno elas saíram, porque a vez que volta na rodada seguinte é outro turno.
type AttackPaid struct {
	CharacterID   *int64 `json:"characterId,omitempty"`
	Scene         int    `json:"scene,omitempty"`
	Round         int    `json:"round,omitempty"`
	Standard      bool   `json:"standard,omitempty"`
	Movement      bool   `json:"movement,omitempty"`
	ExtraMovement bool   `json:"extraMovement,omitempty"`
	// Mana são os recibos do impulso e dos "rolar de novo" (`SheetVitals`).
	Mana []json.RawMessage `json:"mana,omitempty"`
	// HitPoints é o recibo do PV que o gasto do golpe tirou de quem ataca (a Tumular): o cancelamento o
	// devolve de onde saiu, à poça temporária ou ao PV (`SheetVitals`).
	HitPoints json.RawMessage `json:"hitPoints,omitempty"`
}

// AttackDose é a dose em jogo no golpe. Ela viaja escrita porque a faixa só recebe o
// provisório, e o efeito já leva a CD de quem atacou.
type AttackDose struct {
	ItemID    int64  `json:"itemId"`
	CatalogID string `json:"catalogId"`
	Name      string `json:"name"`
	Effect    string `json:"effect"`
	// Spent diz que a dose já saiu da arma: é o que o cancelamento devolve.
	Spent bool `json:"spent,omitempty"`
}

// ForeseenCondition é uma condição que o livro reserva ao golpe, do jeito que a
// mesa lê.
//
// Ela repete os campos do `engine.AnnouncedCondition` em vez de usá-lo, e a
// razão é a FRONTEIRA: o regime não conhece o motor, e o guarda do pacote
// recusa o import (o `ManeuverRoll` logo abaixo existe pela mesma razão). O que
// atravessa é DADO, copiado por quem monta o provisório.
type ForeseenCondition struct {
	// Condition é o id no catálogo de condições; quem o traduz é a cena.
	Condition string `json:"condition"`
	// Note é o que o verbete escreve ao lado — duração, teste que evita.
	Note string `json:"note,omitempty"`
	// Effect é o que o livro prevê e não é condição (o 4d4 de ácido da Corrosiva), com a fórmula escrita;
	// com ele, Condition fica vazia.
	Effect string `json:"effect,omitempty"`
}

// ManeuverRoll é o teste OPOSTO de uma manobra, do jeito que a mesa lê.
//
// Os números de quem ATACA não se repetem aqui: o `Roll` e o `Total` do
// provisório já são dele. O que falta é o outro lado, e é isso que este tipo
// acrescenta — a mesma razão pela qual a `Defense` viaja num golpe: a mesa lê a
// COMPARAÇÃO, e um total sozinho não explica o veredito.
type ManeuverRoll struct {
	// Kind é a manobra do livro: agarrar, derrubar, desarmar, empurrar, quebrar.
	Kind string `json:"kind"`
	// OpposedRoll é o d20 NATURAL de quem se defende, e Opposed é o total dele
	// já com o Luta.
	//
	// Os dois viajam porque a mesa lê a CONTA: "14 (d20 8 +6) contra 12 (d20 4
	// +8)" diz quem teve sorte e quem tem perícia, e dois totais sozinhos não.
	// O natural também é o único sentinela honesto para "este lado foi
	// rolado" — um TOTAL pode ser zero de verdade, e é: um arcanista sem
	// proficiência na espada ataca com −5 (p142), e um d20 de 5 dá zero.
	OpposedRoll int `json:"opposedRoll"`
	Opposed     int `json:"opposed"`
	// Margin é a diferença, e ela é REGRA: cinco pontos ou mais dão efeito extra
	// ao derrubar e ao desarmar (p234). Negativa quando quem tentou perdeu.
	Margin int `json:"margin"`
	// ConditionOnAWin é a condição que o LIVRO diz que a vitória deixa (p234), e
	// ela viaja SEMPRE — não só quando alguém vence, porque o sistema não sabe
	// quem venceu e não é dele dizer.
	//
	// Ela substituiu o `Imposes`, que a confirmação do ataque aplicava sozinha.
	// O conhecimento do livro não sumiu: ele mudou de APLICAÇÃO para INFORMAÇÃO,
	// e a faixa o entrega ao mestre junto do gesto que já existe para marcar
	// condição. É a regra da raiz — o sistema INFORMA, o mestre DECIDE.
	ConditionOnAWin string `json:"conditionOnAWin,omitempty"`
}

// As três decisões do mestre sobre um golpe.
const (
	RulingMiss     = "miss"
	RulingHit      = "hit"
	RulingCritical = "critical"
)

// DamageToRoll é a fórmula do dano depois da decisão, na ordem em que os dados
// se leem: a arma primeiro, os extras depois. Repete o `engine.DamageFormula`
// pela razão do `ForeseenCondition`: o regime não conhece o motor.
type DamageToRoll struct {
	Dice  []DiceGroup `json:"dice"`
	Bonus int         `json:"bonus"`
}

// DiceGroup são os dados de uma parcela: os 3d8 do machado, o 1d6 de fogo.
type DiceGroup struct {
	Count int    `json:"count"`
	Faces int    `json:"faces"`
	Type  string `json:"type,omitempty"`
}

// RolledGroup é uma parcela de dado extra já rolada.
type RolledGroup struct {
	Dice  []int  `json:"dice"`
	Faces int    `json:"faces"`
	Type  string `json:"type,omitempty"`
}

// Attacker descreve quem está agindo sobre o provisório.
type Attacker struct {
	UserID int64
	Role   string
}

// ChosenKindRead é o tipo escolhido no encanto e o que a mesa sabia do alvo: o tipo, do bestiário e do bloco
// do mestre, e a raça, do personagem. Vazio quando a linha não diz.
type ChosenKindRead struct {
	Chosen     string `json:"chosen"`
	TargetKind string `json:"targetKind,omitempty"`
	TargetRace string `json:"targetRace,omitempty"`
}
