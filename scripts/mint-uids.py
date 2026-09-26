#!/usr/bin/env python3
"""Cunha o `uid` — o identificador ESTÁVEL e OPACO de cada verbete (ALE-402).

Por que dois identificadores
-----------------------------
O `id` de hoje é o nome em kebab-case: `esquiva`, `bencao-do-mana`. Ele é
legível, e é o que a URL mostra e o que o `grep` acha — isso fica. O problema é
que ele MUDA quando o nome muda, e o banco guarda referências em dezessete
colunas de nove tabelas, seis delas dentro de JSON e duas dentro de um TermID
composto. Renomear vira migração, e migração de dezessete lugares sai pela
metade.

O `uid` resolve isso sendo a identidade que NUNCA muda. O `id` continua sendo o
nome público; o `uid` passa a ser o que o banco guarda e o que um catálogo usa
para apontar para outro.

Decisão do dono (ALE-402), contra a recomendação medida de manter um id só: o
histórico tem UM renome em 1057 nomes, e ele era um hífen. Fica registrado que
o custo aceito é um campo a mais em 1054 verbetes e uma segunda grafia por
conceito.

A forma, e o que ela protege
-----------------------------
`<prefixo>_<dez caracteres base32>`, do gerador criptográfico. O prefixo diz a
ESPÉCIE (`pwr`, `itm`, `cnd`…) e existe só para orientar quem lê um diff — ele
não deriva do nome, porque um uid derivável não é estável: bastaria alguém
"corrigir" a derivação para ele mudar.

Dez caracteres base32 são 50 bits. Para os 1054 verbetes de hoje a chance de
colisão é desprezível, e de todo modo a cunhagem RECUSA colidir em vez de
confiar na conta.

O que ela nunca faz
-------------------
**Não toca um `uid` que já existe.** Cunhar é operação de nascimento; se ela
pudesse recunhar, o identificador deixaria de ser estável e todo o desenho cai.
O guarda `TestNoUidEverChanges` prende isso com uma linha de base.

E o uid é por CONCEITO, não por linha: o `divine-powers.json` tem 80 linhas
para 72 poderes — "Coragem Total" aparece quatro vezes, uma por deus que a
concede, com o texto byte a byte igual. As quatro dividem um uid, senão o mesmo
poder teria quatro identidades e o apontamento não resolveria nada.

Uso: `python3 scripts/mint-uids.py` (relatório) ou `--aplicar`.
"""
import json
import pathlib
import re
import unicodedata
import secrets
import sys

RAIZ = pathlib.Path(__file__).resolve().parent.parent
DADOS = RAIZ / 'engine-go/domain/catalog/data'

ALFABETO = 'abcdefghijklmnopqrstuvwxyz234567'  # base32 sem 0/1/8/9, que se confundem
TAMANHO = 10

# ONDE moram os verbetes de cada catálogo, declarado ARQUIVO A ARQUIVO.
#
# Duas tentativas anteriores usaram um predicado esperto — "objeto com nome e
# descrição" — e as duas mediram o conjunto errado: a primeira deixou de fora
# os 126 itens que têm `"modifiers": []` (lista vazia é falsa em Python e
# presente em Go, e o guarda acusou a divergência), a segunda varreria 411
# ativações cujo id é DERIVADO do poder e que portanto não têm identidade
# própria.
#
# A declaração explícita é mais longa e não erra: cada linha diz a espécie e o
# CAMINHO até os verbetes. Arquivo fora desta tabela não é cunhado, e o que
# tiver verbete e não estiver aqui sai NOMEADO no relatório.
#
# `caminho` é uma lista de passos a partir da raiz: `[]` é a própria coleção,
# e um nome de campo desce para a coleção aninhada daquele campo.
ESCOPO = {
    'class-powers.json':    ('pwr', [[]]),
    'general-powers.json':  ('pwr', [[]]),
    'tormenta-powers.json': ('pwr', [[]]),
    'granted-powers.json':  ('pwr', [[]]),
    'divine-powers.json':   ('pwr', [[]]),
    'spells.json':          ('spl', [[]]),
    'items.json':           ('itm', [[]]),
    'conditions.json':      ('cnd', [[]]),
    'effect-types.json':    ('eft', [[]]),
    'spell-schools.json':   ('sch', [[]]),
    'expertises.json':      ('exp', [[]]),
    'classes.json':         ('cls', [[]]),
    'gods.json':            ('god', [[]]),
    'races.json':           ('rac', [[]]),
    'bestiary.json':        ('cre', [[]]),
    # Estes dois têm verbete ANINHADO que o banco referencia por conta própria:
    # a escolha de benefício de origem e a de habilidade de raça são gravadas
    # na ficha, então cada uma precisa de identidade.
    'origins.json':         ('org', [[], ['benefits']]),
    # O `origins-source.json` é a transcrição do LIVRO das MESMAS 35 origens —
    # itens, perícias e página —, enquanto o `origins.json` traz os benefícios
    # derivados. Dois arquivos, um conceito: eles não ganham uid próprio, eles
    # HERDAM o da origem. Duas identidades para a mesma origem seria o defeito
    # que o uid existe para impedir.
    'origins-source.json':  ('org', []),
    'race-defs.json':       ('rac', [[], ['abilities']]),
}
# Fora do escopo, e por quê — a lista existe para a ausência ser DITA:
#   activations.json   o id é o do poder que a ativação descreve; ela é satélite
#   options.json       listas de strings para a tela, não verbetes
#   gm-tables.json, dungeon-design.json, devotee-terms.json  tabelas de mestre
#   class-expertises.json  é APONTAMENTO — classe para nomes de perícia, e as
#                          perícias já têm identidade em expertises.json
FORA = {'activations.json', 'options.json',
        'gm-tables.json', 'dungeon-design.json', 'devotee-terms.json',
        'class-expertises.json'}

def cunha(usados: set, prefixo: str) -> str:
    while True:
        uid = prefixo + '_' + ''.join(secrets.choice(ALFABETO) for _ in range(TAMANHO))
        if uid not in usados:
            usados.add(uid)
            return uid


def identidade(no: dict) -> str | None:
    """A identidade PÚBLICA do verbete — é ela que diz quais linhas são o MESMO
    conceito e portanto dividem um uid."""
    if isinstance(no.get('id'), str):
        return no['id']
    if isinstance(no.get('name'), str):
        return no['name']
    return None


def colecao(no):
    """Os itens de uma coleção, seja ela lista ou mapa por id."""
    if isinstance(no, list):
        return [x for x in no if isinstance(x, dict)]
    if isinstance(no, dict):
        return [x for x in no.values() if isinstance(x, dict)]
    return []


def verbetes(dados, caminho: list):
    """Os verbetes num CAMINHO declarado: `[]` é a coleção raiz, e cada passo
    desce para a coleção daquele campo."""
    atual = colecao(dados)
    for passo in caminho:
        adiante = []
        for no in atual:
            adiante.extend(colecao(no.get(passo)))
        atual = adiante
    return atual


def espelha_origens(aplicar: bool) -> int:
    """O `origins-source.json` herda o uid da origem correspondente em
    `origins.json`. Casa pela chave, que é o id em kebab dos dois lados."""
    fonte = json.loads((DADOS / 'origins.json').read_text(encoding='utf-8'))
    porChave = {chave_kebab(o['id']): o['uid'] for o in fonte if o.get('uid')}
    alvo = DADOS / 'origins-source.json'
    dados = json.loads(alvo.read_text(encoding='utf-8'))
    n = 0
    for chave, origem in dados.items():
        uid = porChave.get(chave)
        if uid is None or origem.get('uid') == uid:
            continue
        origem['uid'] = uid
        n += 1
    if n and aplicar:
        alvo.write_text(
            json.dumps(dados, ensure_ascii=False, separators=(',', ':')) + '\n',
            encoding='utf-8')
    return n


def chave_kebab(nome: str) -> str:
    s = ''.join(c for c in unicodedata.normalize('NFD', nome.lower())
                if unicodedata.category(c) != 'Mn')
    return re.sub(r'[^a-z0-9]+', '-', s).strip('-')


def main() -> int:
    aplicar = '--aplicar' in sys.argv
    usados, novos, ja, nao_declarados = set(), 0, 0, []

    arquivos = sorted(DADOS.glob('*.json'))
    # Primeira passada: recolhe os uids que JÁ existem, para nunca colidir.
    for arq in arquivos:
        especie_caminhos = ESCOPO.get(arq.name)
        if especie_caminhos is None:
            continue
        dados = json.loads(arq.read_text(encoding='utf-8'))
        for caminho in especie_caminhos[1]:
            for no in verbetes(dados, caminho):
                if isinstance(no.get('uid'), str):
                    usados.add(no['uid'])

    por_arquivo = {}
    for arq in arquivos:
        if arq.name in FORA:
            continue
        if arq.name not in ESCOPO:
            nao_declarados.append(arq.name)
            continue
        especie, caminhos = ESCOPO[arq.name]
        dados = json.loads(arq.read_text(encoding='utf-8'))
        porIdentidade: dict = {}
        mudou, nesta = False, 0
        for caminho in caminhos:
            for no in verbetes(dados, caminho):
                chave = identidade(no)
                if chave is None:
                    continue
                nesta += 1
                if isinstance(no.get('uid'), str):
                    porIdentidade.setdefault(chave, no['uid'])
                    ja += 1
                    continue
                if chave not in porIdentidade:
                    porIdentidade[chave] = cunha(usados, especie)
                    novos += 1
                no['uid'] = porIdentidade[chave]
                mudou = True
        por_arquivo[arq.name] = (nesta, len(porIdentidade))
        if mudou and aplicar:
            arq.write_text(
                json.dumps(dados, ensure_ascii=False, separators=(',', ':')) + '\n',
                encoding='utf-8')

    # A HERANÇA entre arquivos que descrevem o mesmo conceito. Hoje é um caso
    # só, e ele é declarado aqui em vez de deduzido: "mesmo nome, mesmo uid"
    # como regra geral juntaria a perícia Cura com o efeito Cura.
    herdados = espelha_origens(aplicar)

    for nome, (linhas, conceitos) in sorted(por_arquivo.items()):
        extra = f'  ({linhas - conceitos} linhas dividem uid)' if linhas != conceitos else ''
        print(f'  {conceitos:>4} conceitos em {linhas:>4} linhas  {nome}{extra}')
    print(f'\nuids herdados por origins-source: {herdados}')
    print(f'uids já existentes: {ja} | cunhados agora: {novos} | '
          f'{"GRAVADO" if aplicar else "simulação (use --aplicar)"}')
    # A ausência é DITA: arquivo de catálogo que ninguém declarou nem excluiu
    # sai nomeado, em vez de simplesmente não ser cunhado.
    for nome in nao_declarados:
        print(f'  NÃO DECLARADO {nome}: nem em ESCOPO nem em FORA — decida e volte')
    return 1 if nao_declarados else 0


if __name__ == '__main__':
    raise SystemExit(main())
