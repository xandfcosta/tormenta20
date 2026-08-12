/**
 * The system clipboard behind a seam this project owns: components take a copy
 * function as a prop and the wiring passes this one, so a test asserts what
 * would have been copied without touching `navigator`.
 *
 * @example <InviteDialog onCopy={copyToClipboard} onRotate={rotate} />
 */
export async function copyToClipboard(text: string): Promise<void> {
  await navigator.clipboard.writeText(text)
}
