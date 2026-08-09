import { createFileRoute } from '@tanstack/react-router'
import { CampaignDetailPage } from '@/pages/campaigns/campaign-detail-page'

// Campaign detail at exactly `/campaigns/$id`. Data is prefetched by the
// parent layout route; this just renders the screen.
export const Route = createFileRoute('/campaigns/$id/')({
  // Active tab lives in the URL so deep-links + back button keep context (ALE-29).
  validateSearch: (search: Record<string, unknown>): { tab?: string } =>
    typeof search.tab === 'string' ? { tab: search.tab } : {},
  component: CampaignDetailPage,
})
