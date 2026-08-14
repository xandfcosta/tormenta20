import { For, Match, Switch, createMemo } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { type Block, type Inline, parseMarkdown } from '@/shared/lib/markdown'
import { cn } from '@/shared/lib/utils'

/**
 * Mostra o markdown das notas do mestre (ALE-122).
 *
 * Renderiza a ÁRVORE do `parseMarkdown` como elementos Solid — nunca
 * `innerHTML`. Sem HTML no caminho não existe injeção para sanitizar, e o
 * projeto não ganha um parser de terceiro só para escrever uma lista.
 *
 * @example <MarkdownView source={session.notes} />
 */
export function MarkdownView(props: { source: string; class?: string }) {
  const blocks = createMemo(() => parseMarkdown(props.source))

  return (
    <div class={cn('space-y-2 text-sm text-muted-foreground', props.class)}>
      <For each={blocks()}>{(block) => <BlockView block={block} />}</For>
    </div>
  )
}

const HEADING_CLASS: Record<1 | 2 | 3, string> = {
  1: 'text-base',
  2: 'text-sm',
  3: 'text-xs',
}

function BlockView(props: { block: Block }) {
  return (
    <Switch>
      <Match when={props.block.kind === 'heading' && props.block}>
        {(heading) => (
          <Dynamic
            component={`h${heading().level}`}
            class={cn(
              'font-heading uppercase tracking-wide text-grimorio-gold',
              HEADING_CLASS[heading().level],
            )}
          >
            <Spans spans={heading().spans} />
          </Dynamic>
        )}
      </Match>
      <Match when={props.block.kind === 'paragraph' && props.block}>
        {(paragraph) => (
          <p>
            <Spans spans={paragraph().spans} />
          </p>
        )}
      </Match>
      <Match when={props.block.kind === 'list' && props.block}>
        {(list) => (
          <Dynamic
            component={list().ordered ? 'ol' : 'ul'}
            class={cn('ml-5 space-y-1', list().ordered ? 'list-decimal' : 'list-disc')}
          >
            <For each={list().items}>
              {(item) => (
                <li>
                  <Spans spans={item} />
                </li>
              )}
            </For>
          </Dynamic>
        )}
      </Match>
      <Match when={props.block.kind === 'quote' && props.block}>
        {(quote) => (
          <blockquote class="border-l-2 border-grimorio-iron pl-3 italic">
            <Spans spans={quote().spans} />
          </blockquote>
        )}
      </Match>
      <Match when={props.block.kind === 'rule'}>
        <hr class="border-grimorio-iron" />
      </Match>
    </Switch>
  )
}

function Spans(props: { spans: Inline[] }) {
  return <For each={props.spans}>{(span) => <SpanView span={span} />}</For>
}

function SpanView(props: { span: Inline }) {
  return (
    <Switch fallback={props.span.text}>
      <Match when={props.span.kind === 'strong'}>
        <strong class="font-semibold text-foreground">{props.span.text}</strong>
      </Match>
      <Match when={props.span.kind === 'em'}>
        <em>{props.span.text}</em>
      </Match>
      <Match when={props.span.kind === 'code'}>
        <code class="rounded-xs bg-grimorio-iron/40 px-1 font-mono text-xs">{props.span.text}</code>
      </Match>
      <Match when={props.span.kind === 'link' && props.span}>
        {(link) => (
          <a
            href={link().href}
            target="_blank"
            rel="noreferrer noopener"
            class="text-[color:var(--primary)] underline"
          >
            {link().text}
          </a>
        )}
      </Match>
    </Switch>
  )
}
