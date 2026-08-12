import { createFileRoute } from '@tanstack/solid-router'
import { z } from 'zod'
import { CampaignJoinPage } from '@/pages/campaigns/campaign-join-page'
import { requireSession } from './-guards'

// The invite token rides in the URL so the link the GM shares is the whole
// state — reload it, bookmark it, log in and come back, it still works.
const searchSchema = z.object({ token: z.string().optional() })

export const Route = createFileRoute('/campaigns/join')({
  validateSearch: searchSchema,
  beforeLoad: requireSession,
  component: CampaignJoinPage,
})
