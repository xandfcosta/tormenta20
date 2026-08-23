import { z } from 'zod'

/**
 * The rules a chronicle's name and description obey — shared by "abrir nova
 * campanha" and "editar campanha" so the two forms can never disagree about
 * what a valid campaign is (ALE-80).
 *
 * `.trim()` runs before `.min(1)`, so a name of pure spaces is rejected
 * instead of leaving the chronicle untitled in the book.
 *
 * @example campaignSchema.safeParse({ name: '  ', description: '' }) // → erro no nome
 */
export const campaignSchema = z.object({
  name: z.string().trim().min(1, 'Nome é obrigatório').max(120, 'Máximo 120 caracteres'),
  description: z.string().max(2000, 'Máximo 2000 caracteres'),
})

export type CampaignFormValues = z.infer<typeof campaignSchema>
