import { createFileRoute } from '@tanstack/solid-router'
import { CampaignsListPage } from '@/pages/campaigns/campaign-list-page'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/')({
  beforeLoad: requireSession,
  component: CampaignsListPage,
})
