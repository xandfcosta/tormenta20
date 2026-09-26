#!/usr/bin/env python3
"""Confere as CONDIÇÕES contra o livro (p394-395). Seção da ALE-391.

O que este auditor mede, e onde o dado mora
--------------------------------------------
As condições são o único catálogo cujo NÚMERO não está no JSON. O
`conditions.json` tem nome, descrição em prosa e página; a REGRA mora no
`conditionModifierTable` do `domain/engine/collect.go`, escrita em Go.

O que se mede é a HERANÇA: quase todo verbete do livro começa dizendo de quais
outras condições ele herda o efeito — "Enredado. O personagem fica lento,
vulnerável e sofre –2 em testes de ataque." A tabela do motor declara a mesma
coisa referenciando `lentoMods`, `vulneravelMods` e companhia, então as duas
são comparáveis conjunto a conjunto.

É a dimensão que mais rendeu: foi assim que o "e lento" do Cego ficou anos de
fora (ALE-390), e foi assim que o Agarrado e o Enredado foram achados — o
conserto do Cego não varreu os irmãos dele.

Ler Go dá trabalho, e ler MAL dá um defeito inventado
------------------------------------------------------
A primeira versão lia uma linha por entrada e perdia a continuação: o Cego, que
ocupa duas linhas, saiu herdando só `desprevenido` e teria sido acusado de um
defeito que não tem. A leitura é por PROFUNDIDADE de parêntese, da chave até a
vírgula de nível zero, e o parser **falha no identificador que não conhece** em
vez de descartá-lo — uma lista de proibidos subconta em silêncio.

Os dois falsos positivos são NOMEADOS
--------------------------------------
Nem toda diferença é defeito, e as duas que não são estão na `ACEITAS`:

- **Indefeso** — o livro diz "fica desprevenido, MAS sofre −10 na Defesa". O
  "mas" faz o −10 SUBSTITUIR o −5 do desprevenido, então não herdar é o certo.
- **Petrificado** — o livro manda herdar de `inconsciente`, que por sua vez é
  `indefeso`. O motor achata a cadeia e chega no mesmo lugar.

Uma terceira diferença passa a reprovar, que é o que uma lista de permitidos
dá e uma de proibidos não.

Uso: `python3 scripts/audit-conditions.py`. Precisa do PDF do livro (t20pdf).
"""
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import RAIZ, junta, linhas_com_coordenada, sem_lixo  # noqa: E402

PRIMEIRA, ULTIMA = 400, 401  # PDF; livro = PDF - 6
CONDICOES = RAIZ / 'engine-go/domain/catalog/data/conditions.json'
TABELA = RAIZ / 'engine-go/domain/engine/collect.go'
DECLARACAO = 'var conditionModifierTable = map[string][]Modifier{'

# Variável de modificadores → a condição que ela representa. É o vocabulário do
# lado do motor, e o que não estiver aqui REPROVA.
HERANCA = {
    'lentoMods': 'lento', 'imovelMods': 'imovel', 'vulneravelMods': 'vulneravel',
    'desprevenidoMods': 'desprevenido', 'indefesoMods': 'indefeso',
    'fracoMods': 'fraco', 'debilitadoMods': 'debilitado',
}
# Os ajudantes que a tabela pode chamar. Identificador fora destes dois
# conjuntos faz o auditor parar: é sinal de que a tabela ganhou uma forma que
# ele não sabe ler, e seguir em frente produziria uma lista de falhas com cara
# de descoberta (ALE-294).
AJUDANTES = {
    'juntar', 'append', 'Modifier', 'condAllSkills', 'condIntSabCar', 'condAttack',
    'condSkill', 'condByAttr', 'condDefense', 'condDefenseVs', 'condDamageReduction',
    'condPmCost', 'condFactor', 'condFlag', 'condMod', 'ModifierTarget', 'K', 'Name',
    'Scope', 'Attribute',
}
ACEITAS = {
    'indefeso': ('o livro diz "fica desprevenido, MAS sofre −10 na Defesa" — '
                 'o "mas" SUBSTITUI o −5, então não herdar é o certo'),
    'petrificado': ('o livro manda herdar de `inconsciente`, que é `indefeso` — '
                    'o motor achata a cadeia e chega no mesmo lugar'),
}


def blocos_do_livro(condicoes: dict) -> dict:
    """{id: texto do verbete}, cortado no início do verbete seguinte."""
    partes = []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        ordenadas = sorted(linhas_com_coordenada(pagina), key=lambda r: (r[0] // 200, r[1]))
        partes.append(junta([t for _x, _y, t in sem_lixo(ordenadas)]))
    inteiro = ' '.join(partes)
    # O livro escreve o nome em Title Case seguido de ponto — "Enredado. ".
    cortes = sorted((m.start(), id_) for id_, c in condicoes.items()
                    for m in re.finditer(re.escape(c['name']) + r'\. ', inteiro))
    blocos = {}
    for (i, id_), (j, _) in zip(cortes, cortes[1:] + [(len(inteiro), None)]):
        blocos.setdefault(id_, inteiro[i:j])
    return blocos


def herda_no_livro(bloco: str, id_: str, por_nome: dict) -> set:
    """As condições que a PRIMEIRA frase do verbete diz herdar.

    Só a primeira frase, e a razão é o Fatigado: a segunda diz "Se ficar
    fatigado novamente, em vez disso fica exausto", que é AGRAVAMENTO e não
    herança — lê-la junto faria toda condição que piora herdar a pior.
    """
    primeira = bloco.split('. ', 1)[1].split('.')[0].lower()
    return {outro for nome, outro in por_nome.items()
            if re.search(r'\b' + re.escape(nome) + r'\b', primeira) and outro != id_}


def le_a_tabela() -> tuple[dict, list[str]]:
    """{id: expressão} de cada entrada, lida por profundidade de parêntese."""
    fonte = TABELA.read_text(encoding='utf-8')
    corpo = fonte[fonte.index('{', fonte.index(DECLARACAO)) + 1:]
    entradas, queixas, pos = {}, [], 0
    while (m := re.compile(r'"([a-z-]+)":\s*').search(corpo, pos)):
        fim, profundidade = m.end(), 0
        while fim < len(corpo):
            c = corpo[fim]
            if c in '({[':
                profundidade += 1
            elif c in ')}]':
                if profundidade == 0:
                    break
                profundidade -= 1
            elif c == ',' and profundidade == 0:
                break
            fim += 1
        entradas[m.group(1)] = corpo[m.end():fim]
        pos = fim + 1
        if corpo[fim:fim + 1] in ')}]':
            break
    for id_, expr in entradas.items():
        # O ARGUMENTO não é identificador: `condSkill("Iniciativa", -2)` traz o
        # nome de uma perícia, e `condDefenseVs("melee", …)` traz um escopo.
        # Varrer dentro das aspas faria o auditor parar em quatro nomes
        # legítimos e não medir nada.
        sem_strings = re.sub(r'"[^"]*"', '""', expr)
        for palavra in re.findall(r'\b([A-Za-z][A-Za-z0-9_]*)\b', sem_strings):
            if palavra not in HERANCA and palavra not in AJUDANTES:
                queixas.append(f'{id_}: a tabela usa {palavra!r}, que este auditor não '
                               f'sabe ler — ele pararia de medir a herança em silêncio')
    return entradas, queixas


def main() -> int:
    condicoes = json.load(open(CONDICOES, encoding='utf-8'))
    por_nome = {c['name'].lower(): id_ for id_, c in condicoes.items()}
    blocos = blocos_do_livro(condicoes)
    entradas, queixas = le_a_tabela()

    divergem = 0
    for id_ in sorted(condicoes):
        if id_ not in entradas:
            continue
        if id_ not in blocos:
            queixas.append(f'{condicoes[id_]["name"]}: sem verbete no livro')
            continue
        livro = herda_no_livro(blocos[id_], id_, por_nome)
        motor = {HERANCA[v] for v in HERANCA if v in entradas[id_]} - {id_}
        if livro == motor:
            continue
        if id_ in ACEITAS:
            print(f'  DIFERENÇA ACEITA {condicoes[id_]["name"]}: {ACEITAS[id_]}')
            continue
        divergem += 1
        print(f'  {condicoes[id_]["name"]}: o livro diz que ela herda {sorted(livro)} '
              f'e o motor herda {sorted(motor)} — falta {sorted(livro - motor)}')

    sem_regra = sorted(set(condicoes) - set(entradas))
    print(f'\ncondições: {len(condicoes)} | com verbete no livro: {len(blocos)} '
          f'| com regra no motor: {len(entradas)} | divergem: {divergem}')
    print(f'sem efeito mecânico, e por isso fora da tabela ({len(sem_regra)}): '
          f'{", ".join(condicoes[i]["name"] for i in sem_regra)}')
    for queixa in queixas:
        print(f'  NÃO MEDIDO {queixa}')
    return 1 if divergem or queixas else 0


if __name__ == '__main__':
    raise SystemExit(main())
