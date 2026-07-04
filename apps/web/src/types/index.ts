// ── Shared enums ────────────────────────────────────────────

export type Role = "admin" | "business" | "promoter";
export type Platform = "instagram" | "twitter" | "tiktok";
export type CampaignStatus = "draft" | "active" | "paused" | "completed" | "cancelled";
export type SubmissionStatus = "pending" | "approved" | "rejected";
export type WithdrawalStatus = "pending" | "processing" | "completed" | "failed";

// ── User ────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  role: Role;
  name: string;
  avatar?: string;
  isEmailVerified: boolean;
  isSuspended: boolean;
  trustScore: number;
  badges: string[];
  createdAt: string;
  updatedAt: string;
}

// ── Social Account ───────────────────────────────────────────

export interface SocialAccount {
  id: string;
  userId: string;
  platform: Platform;
  platformUserId: string;
  username: string;
  profileUrl: string;
  tier: number;
  isVerified: boolean;
  status: "pending_review" | "active" | "rejected";
  rejectedReason?: string;
  lastSyncedAt: string;
  createdAt: string;
}

// ── Campaign ─────────────────────────────────────────────────

export interface Campaign {
  id: string;
  businessId: string;
  title: string;
  description: string;
  targetUrl: string;
  mediaAssets: string[];
  platform: Platform;
  budget: number;
  remainingBudget: number;
  baseRepostRate: number;
  minFollowers: number;
  minInfluenceScore: number;
  maxParticipants: number;
  currentParticipants: number;
  status: CampaignStatus;
  startDate: string;
  endDate: string;
  createdAt: string;
  updatedAt: string;
}

// ── Submission ───────────────────────────────────────────────

export interface CampaignSubmission {
  id: string;
  campaignId: string;
  promoterId: string;
  businessId: string;
  repostUrl: string;
  screenshotUrl: string;
  status: SubmissionStatus;
  rejectionReason?: string;
  reviewedBy?: string;
  baseAmount: number;
  influenceMultiplier: number;
  finalAmount: number;
  platformFee: number;
  promoterEarning: number;
  submittedAt: string;
  reviewedAt?: string;
  payoutReleasedAt?: string;
  createdAt: string;
  updatedAt: string;
}

// ── Wallet ───────────────────────────────────────────────────

export interface Wallet {
  id: string;
  userId: string;
  role: Role;
  availableBalance: number;
  pendingBalance: number;
  totalEarned: number;
  totalSpent: number;
  currency: string;
  updatedAt: string;
  createdAt: string;
}

export interface Transaction {
  id: string;
  walletId: string;
  userId: string;
  type: string;
  amount: number;
  fee: number;
  balanceAfter: number;
  referenceId: string;
  description: string;
  createdAt: string;
}

export interface Withdrawal {
  id: string;
  userId: string;
  amount: number;
  fee: number;
  netAmount: number;
  payoutId?: string;
  status: WithdrawalStatus;
  requestedAt: string;
  processedAt?: string;
  createdAt: string;
}

// ── Notification ─────────────────────────────────────────────

export interface Notification {
  id: string;
  userId: string;
  type: string;
  title: string;
  message: string;
  isRead: boolean;
  metadata?: Record<string, unknown>;
  createdAt: string;
}

// ── Chat ─────────────────────────────────────────────────────

export interface UserSummary {
  id: string;
  name: string;
  avatar?: string;
  role: Role;
}

export interface Conversation {
  id: string;
  otherParty: UserSummary;
  lastMessageAt: string;
  lastMessagePreview: string;
  unreadCount: number;
  createdAt: string;
}

export interface AdminConversation {
  id: string;
  business: UserSummary;
  promoter: UserSummary;
  lastMessageAt: string;
  lastMessagePreview: string;
  createdAt: string;
}

export interface ChatMessage {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
}

// ── API response wrapper ─────────────────────────────────────

export interface ApiResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  errors?: unknown;
  meta?: {
    total?: number;
    page?: number;
    limit?: number;
  };
}
