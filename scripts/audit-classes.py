#!/usr/bin/env python3
"""Confere as CLASSES do app contra o livro, poder por poder (ALE-392).

Rodar é ATO DELIBERADO, como o `genoracle` e os auditores irmãos: a ferramenta
PROPÕE, o diff se revisa contra o PDF, e é a revisão que decide. Ela não escreve.

    python3 scripts/audit-classes.py              # relatório
    python3 scripts/audit-classes.py --classe Bárbaro

O livro escreve poder de classe de DUAS formas, e elas pedem âncoras diferentes
-------------------------------------------------------------------------------
1. **Prosa** — `• Nome. texto` (escolhível) ou `Nome. texto` (concedida), na
   seção da classe. A âncora é o NOME dentro da faixa de páginas dela.
2. **Degrau de tabela** — só a tabela da classe o nomeia: `6º · Fúria +3, poder
   de bárbaro`. A âncora é o NÍVEL, e ela é a mais forte das duas, porque é um
   número que o catálogo também afirma (`grantedAtLevel`).

Medido: dos 462 poderes, 345 ancoram na prosa e os outros 117 são degraus. E dos
que ancoram na prosa, só 14 têm modificador — os que produzem número são quase
todos degraus. Um auditor só de prosa mediria 14 de 49, que é o verde sobre nada.

A tabela se lê por Y, e a coluna de habilidade se APRENDE
----------------------------------------------------------
O nível e a habilidade são blocos DIFERENTES do PDF; o que diz que "6º" e
"Fúria +3" são a mesma linha é o y das duas ser igual. Parear por índice erra na
primeira habilidade que quebra em duas linhas — e a do Bárbaro quebra no 11º.
A linha que não casa com nível nenhum é CONTINUAÇÃO da anterior.

E a coluna da habilidade não é "tudo à direita do nível": em página cuja tabela
fica no meio, isso apanha a coluna de PROSA. Ela é o x mais FREQUENTE entre as
linhas que pareiam com um nível — aprender custou cinco falsos "não nomeado na
tabela" que pareciam defeito de catálogo.

A heurística de modificador, e o falso positivo que ela tem
------------------------------------------------------------
Para cada modificador, o ALVO dele tem de ser nomeado no texto do livro: um
`expertise:Fortitude +2` pede a palavra "Fortitude" na frase. Foi assim que a
Fúria do bárbaro foi pega concedendo Fortitude e Vontade que o livro não dá
(ALE-388).

Ela tem um falso positivo CONHECIDO: o poder que se define por REFERÊNCIA a
outro. "Pele de Aço. O bônus de Pele de Ferro aumenta para +8" não diz "Defesa",
e modelar como +4 sobre o +4 do Pele de Ferro está certo. Esses saem marcados
como REFERÊNCIA, e não como divergência.

Precisa do `pdftotext` (poppler) e do PDF do livro.
"""
import argparse
import collections
import json
import re
import sys

from t20pdf import RAIZ, chave, linhas_com_coordenada, linhas_da_pagina, normaliza_frase

PODERES = str(RAIZ / 'engine-go/domain/catalog/data/class-powers.json')
# As páginas saem do SUMÁRIO do livro, lido por coordenada. Classes vão de 32 a
# 84 — Origens começam em 85.
PRIMEIRA_PAGINA_DA_CLASSE = {
    'Arcanista': 36, 'Bárbaro': 40, 'Bardo': 43, 'Bucaneiro': 46, 'Caçador': 49,
    'Cavaleiro': 52, 'Clérigo': 56, 'Druida': 60, 'Guerreiro': 64, 'Inventor': 67,
    'Ladino': 72, 'Lutador': 75, 'Nobre': 78, 'Paladino': 81,
}
DEPOIS_DAS_CLASSES = 85
OFFSET = 6  # página do PDF = página do livro + 6

RE_NIVEL = re.compile(r'^(\d{1,2})º$')
RE_ENTRADA = re.compile(r'^(?:•\s*)?([A-ZÁÉÍÓÚÂÊÔÃÕÇ][^.]{2,44})\.\s+(.*)$')
LIXO = re.compile(r'Mateus Santos|mateush\.santos|Construção de Personagem|^\d{1,3}$|^Capítulo')
# O poder que se define por referência a outro: o alvo não é nomeado, e está
# certo que não seja.
RE_REFERENCIA = re.compile(r'\b(b[oô]nus de|aumenta para|passa a ser|em vez disso)\b', re.I)

ATRIBUTO = {'strength': 'Força', 'dexterity': 'Destreza', 'constitution': 'Constituição',
            'intelligence': 'Inteligência', 'wisdom': 'Sabedoria', 'charisma': 'Carisma'}
TERMO_DO_ALVO = {
    'attack': ['ataque'], 'damage': ['dano'], 'defense': ['Defesa'],
    'maxPv': ['pontos de vida', 'PV'], 'maxPm': ['pontos de mana', 'PM'],
    'pmCost': ['PM'], 'spellDC': ['CD'], 'damageReduction': ['redução de dano'],
    'displacement': ['deslocamento'], 'pmLimit': ['PM'], 'tempHp': ['pontos de vida temporários'],
}


def faixas() -> dict:
    """De que página a que página vai cada classe."""
    ordenadas = sorted(PRIMEIRA_PAGINA_DA_CLASSE.items(), key=lambda kv: kv[1])
    saida = {}
    for i, (nome, p) in enumerate(ordenadas):
        ate = ordenadas[i + 1][1] - 1 if i + 1 < len(ordenadas) else DEPOIS_DAS_CLASSES - 1
        saida[nome] = (p, ate)
    return saida


def tabela_da_classe(classe: str, primeira: int, ultima: int) -> dict:
    """{nível: texto da linha} da tabela da classe, pareado por Y."""
    for pagina in range(primeira + OFFSET, ultima + OFFSET + 1):
        linhas = [l for l in linhas_com_coordenada(pagina) if not LIXO.search(l[2])]
        titulo = [l for l in linhas
                  if l[2].startswith('Tabela') and chave(classe) in chave(l[2])]
        if not titulo:
            continue
        corpo = [l for l in linhas if l[1] > titulo[0][1]]
        niveis = {round(y): int(RE_NIVEL.match(t).group(1))
                  for x, y, t in corpo if RE_NIVEL.match(t)}
        if not niveis:
            continue
        x_do_nivel = max(x for x, y, t in corpo if RE_NIVEL.match(t))
        candidatas = [(x, y, t) for x, y, t in corpo if x > x_do_nivel + 5]
        casam = [x for x, y, t in candidatas if any(abs(yy - y) <= 2 for yy in niveis)]
        if not casam:
            continue
        x_da_habilidade = collections.Counter(round(x, 1) for x in casam).most_common(1)[0][0]

        saida, atual = {}, None
        for x, y, t in sorted((l for l in candidatas if abs(l[0] - x_da_habilidade) <= 3),
                              key=lambda l: l[1]):
            casa = [n for yy, n in niveis.items() if abs(yy - y) <= 2]
            if casa:
                atual = casa[0]
                saida[atual] = t
            elif atual is not None:
                saida[atual] += ' ' + t
        return saida
    return {}


def prosa_da_classe(primeira: int, ultima: int) -> dict:
    """{chave do nome: [linhas do corpo]} das entradas em prosa."""
    linhas = []
    for pagina in range(primeira + OFFSET, ultima + OFFSET + 1):
        linhas += linhas_da_pagina(pagina)
    saida, atual = {}, None
    for linha in linhas:
        m = RE_ENTRADA.match(linha.strip())
        if m:
            atual = chave(m.group(1))
            saida.setdefault(atual, [m.group(2)])
        elif atual and atual in saida:
            saida[atual].append(linha.strip())
    return saida


def termos_do_alvo(modificador: dict) -> list:
    alvo = modificador.get('target') or {}
    if alvo.get('k') == 'expertise':
        return [alvo.get('name', '')]
    if alvo.get('k') == 'attribute':
        return [ATRIBUTO.get(alvo.get('attribute'), '')]
    return TERMO_DO_ALVO.get(alvo.get('k'), [])


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('--classe', help='auditar só uma')
    args = ap.parse_args()

    faixa = faixas()
    alvo = [args.classe] if args.classe else list(faixa)
    with open(PODERES, encoding='utf-8') as f:
        bruto = json.load(f)
    poderes = bruto if isinstance(bruto, list) else list(bruto.values())

    tabelas = {c: tabela_da_classe(c, *faixa[c]) for c in alvo}
    prosas = {c: prosa_da_classe(*faixa[c]) for c in alvo}

    for c in alvo:
        if len(tabelas[c]) != 20:
            print(f'ATENÇÃO {c}: a tabela leu {len(tabelas[c])} níveis e o livro tem 20 — '
                  f'o relatório abaixo está medindo menos do que diz.')

    nivelOK = nivelDiverge = semTabela = 0
    naProsa = semAncora = 0
    divergentes, referencias = [], []
    for p in poderes:
        c = p.get('className')
        if c not in alvo:
            continue
        k = chave(p['name'])

        if p.get('grantedAtLevel'):
            achados = [n for n, txt in tabelas[c].items() if k in chave(txt)]
            if not achados:
                semTabela += 1
            elif p['grantedAtLevel'] in achados:
                nivelOK += 1
            else:
                nivelDiverge += 1
                divergentes.append(('NÍVEL', c, p['name'],
                                    f"catálogo={p['grantedAtLevel']} livro={achados}"))

        corpo = prosas[c].get(k)
        if corpo is None:
            semAncora += 1
            continue
        naProsa += 1
        texto = ' '.join(corpo)
        normalizado = normaliza_frase(texto)
        for m in p.get('modifiers') or []:
            termos = [t for t in termos_do_alvo(m) if t]
            if not termos or any(normaliza_frase(t) in normalizado for t in termos):
                continue
            onde = referencias if RE_REFERENCIA.search(texto) else divergentes
            onde.append(('ALVO', c, p['name'],
                         f"{(m.get('target') or {}).get('k')} '{termos[0]}' "
                         f"{m.get('amount'):+d} não é nomeado no livro"))

    comNivel = sum(1 for p in poderes if p.get('className') in alvo and p.get('grantedAtLevel'))
    doAlvo = [p for p in poderes if p.get('className') in alvo]
    print(f'poderes no catálogo:        {len(doAlvo)}')
    print(f'  ancorados na prosa:       {naProsa}')
    print(f'  sem âncora na prosa:      {semAncora}  (degraus de tabela, na maioria)')
    print(f'com grantedAtLevel:         {comNivel}')
    print(f'  nível confere:            {nivelOK}')
    print(f'  nível DIVERGE:            {nivelDiverge}')
    print(f'  não nomeado na tabela:    {semTabela}')
    print(f'\ndivergências de ALVO/NÍVEL: {len(divergentes)}')
    for tipo, c, nome, o_que in divergentes:
        print(f'  {tipo:6s} {c:11s} {nome[:34]:36s} {o_que}')
    print(f'\npor REFERÊNCIA a outro poder (falso positivo conhecido): {len(referencias)}')
    for _tipo, c, nome, o_que in referencias:
        print(f'         {c:11s} {nome[:34]:36s} {o_que}')


if __name__ == '__main__':
    sys.exit(main())
