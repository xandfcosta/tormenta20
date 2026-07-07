import { IsIn, IsInt, IsOptional, IsString, MinLength, Min } from 'class-validator';

export const CAMPAIGN_MEMBER_ROLES = ['player', 'gm'] as const;
export type CampaignMemberRole = (typeof CAMPAIGN_MEMBER_ROLES)[number];

export class AddMemberDto {
  @IsInt()
  @Min(1)
  characterId!: number;

  @IsOptional()
  @IsIn(CAMPAIGN_MEMBER_ROLES)
  role?: CampaignMemberRole;

  /**
   * Optional invite token. When present the join goes through the
   * invite path: caller must own the character (unchanged) AND the
   * token must match `campaignId`. When absent the classic self-join
   * rules apply.
   */
  @IsOptional()
  @IsString()
  @MinLength(8)
  inviteToken?: string;
}

export class UpdateMemberDto {
  @IsIn(CAMPAIGN_MEMBER_ROLES)
  role!: CampaignMemberRole;
}
