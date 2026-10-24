#!/usr/bin/env python3
"""O que o MOTOR aplica de cada verbete de raça, classe e origem (ALE-482).

    python3 scripts/audit-coverage.py              # os três perfis
    python3 scripts/audit-coverage.py classe       # um só
    python3 scripts/audit-coverage.py --destino    # só a contagem por issue

Não é um auditor do LIVRO
-------------------------
Os irmãos (`audit-races.py`, `audit-classes.py`, `audit-origins.py`) conferem o
catálogo contra a página. Este confere o catálogo contra o MOTOR: de cada
verbete, quanto vira número, quanto vira gesto e quanto fica com o mestre. Por
isso ele não precisa do PDF, e a resposta dele muda quando o motor muda.

O denominador mora no DADO
--------------------------
A cobertura do capítulo 1 foi contada quatro vezes em duas sessões e saiu errada
nas quatro, sempre por um instrumento que inferia o balde de fora — a última
contou as 125 passivas sem `grant` como gesto. O balde agora é DECLARADO no
verbete, nos mesmos termos dos encantos (ALE-459):

* tem efeito no motor (`modifiers`, variante com `modifiers`, `powerUid`, ou
  ativação com `grant`) ou `modeled` dizendo por onde ele entra → INTEIRO;
* tem efeito E `unmodeled` → EM PARTE, e aí o `modeled` é obrigatório: sem ele a
  nota parece falar do verbete inteiro;
* ativação que não é `passive` e não tem `grant` → GESTO: o app cobra o PM e
  anuncia, e quem aplica é o mestre;
* nada disso → SÓ NOTA, e o `unmodeled` é obrigatório.

"Inteiro" quer dizer "tem efeito e ninguém declarou falta" — o mesmo limite do
auditor de encantos. Se uma metade não entrou, quem a nota é a revisão, e a
nota passa a sair aqui.

O DESTINO é a issue que a nota cita
-----------------------------------
Toda nota que tem conserto cita a fatia que o resolve (`ALE-NNN`). Agrupar por
ela é o que transforma esta lista no plano do capítulo: a fatia sabe quantos
verbetes a esperam, com os nomes. A nota que cita DUAS sai nas duas, e por isso
a soma dos grupos passa do número de notas. A nota que não cita issue é prosa do mestre ou
forma sem fatia, e as duas saem no grupo "sem issue" para ninguém as perder.
"""
import collections
import json
import pathlib
import re
import sys

DADOS = pathlib.Path(__file__).resolve().parent.parent / 'engine-go/domain/catalog/data'
RE_ISSUE = re.compile(r'ALE-\d+')

# O benefício de PERÍCIA da origem não tem marca por verbete: são 94, e todos
# entram pela mesma forma. A regra vale para o TIPO, e mora aqui uma vez.
PERICIA_DE_ORIGEM = 'treina a perícia pelo modificador `trained` que o `originModifiers` gera (ALE-490)'


def ler(nome):
    return json.loads((DADOS / f'{nome}.json').read_text(encoding='utf-8'))


ATIVACOES = {a['id']: a for a in ler('activations')}


def balde(verbete):
    """(balde, falha ou None) de um verbete."""
    ativacao = ATIVACOES.get(verbete['id'])
    efeito = bool(
        verbete.get('modifiers') or verbete.get('powerUid')
        or any(v.get('modifiers') for v in verbete.get('variants', []))
        or (ativacao and ativacao.get('grant')))
    modelado, motivo = verbete.get('modeled'), verbete.get('unmodeled')
    if (efeito or modelado) and motivo:
        if not modelado:
            return 'em parte', 'aplica alguma coisa e não diz O QUÊ (`modeled`)'
        return 'em parte', None
    if efeito or modelado:
        return 'inteiro', None
    if ativacao and ativacao['kind'] != 'passive':
        return 'gesto', None
    if motivo:
        return 'só nota', None
    return 'só nota', 'não tem efeito no motor NEM diz por quê (`unmodeled`)'


def verbetes(perfil):
    """(rótulo, verbete) de cada verbete do perfil."""
    if perfil == 'raça':
        for raca in ler('race-defs'):
            for h in raca['abilities']:
                yield f'{raca["name"]}: {h["name"]}', h
    elif perfil == 'classe':
        for p in ler('class-powers'):
            yield f'{p.get("className", "(genérico)")}: {p.get("name", p["id"])}', p
    elif perfil == 'origem':
        for o in ler('origins'):
            yield f'{o["name"]}: {o["poderUnico"]["name"]}', o['poderUnico']


def beneficios_de_origem():
    """Contagem dos benefícios de origem por tipo — o denominador do resto."""
    tipos = collections.Counter()
    for o in ler('origins'):
        for b in o['benefits']:
            tipos['aponta para poder geral' if b.get('powerUid')
                  else 'poder livre' if b.get('powerPick')
                  else b['kind']] += 1
    return tipos


def audita(perfil, por_destino):
    contagem = collections.Counter()
    falhas = []
    for rotulo, v in verbetes(perfil):
        b, falha = balde(v)
        contagem[b] += 1
        if falha:
            falhas.append(f'{rotulo}: {falha}')
        motivo = v.get('unmodeled')
        if motivo:
            # Em TODA issue que ela cita: a do Lefou é tipo de criatura (ALE-469)
            # e efeito condicional (ALE-484), e cada fatia tem de achá-la.
            for destino in sorted(set(RE_ISSUE.findall(motivo))) or ['sem issue']:
                por_destino[destino].append((perfil, rotulo, b, motivo))
    total = sum(contagem.values())
    print(f'\n{perfil.upper()} — {total} verbetes')
    for b in ('inteiro', 'em parte', 'gesto', 'só nota'):
        print(f'  {b:9s} {contagem[b]:4d}')
    if perfil == 'origem':
        tipos = beneficios_de_origem()
        print(f'  e os {sum(tipos.values())} benefícios: ' + ', '.join(f'{n} {t}' for t, n in tipos.most_common()))
        print(f'    perícia: {PERICIA_DE_ORIGEM}')
    print(f'  FALHAS: {len(falhas)}   (medidos: {total})')
    for f in falhas:
        print(f'    {f}')
    return len(falhas)


def main():
    argumentos = [a for a in sys.argv[1:] if not a.startswith('--')]
    so_destino = '--destino' in sys.argv
    perfis = argumentos or ['raça', 'classe', 'origem']
    desconhecidos = [p for p in perfis if p not in ('raça', 'classe', 'origem')]
    if desconhecidos:
        print(f'não conheço {desconhecidos}; os perfis são raça, classe e origem')
        return 1
    por_destino = collections.defaultdict(list)
    falhas = sum(audita(p, por_destino) for p in perfis)
    print('\nPOR DESTINO — a nota de cada verbete, agrupada pela issue que ela cita')
    for destino in sorted(por_destino, key=lambda d: (d == 'sem issue', d)):
        linhas = por_destino[destino]
        print(f'\n  {destino}: {len(linhas)}')
        if so_destino:
            continue
        for perfil, rotulo, b, motivo in linhas:
            print(f'    [{perfil}] {rotulo}  ({b})')
            print(f'        {motivo}')
    return 1 if falhas else 0


if __name__ == '__main__':
    sys.exit(main())
