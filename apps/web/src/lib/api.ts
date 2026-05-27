import api from "./axios";
import type {
  User,
  Campaign,
  SocialAccount,
  CampaignSubmission,
  Wallet,
  Transaction,
  Withdrawal,
  Notification,
} from "@/types";

// ── Helpers ──────────────────────────────────────────────────

type R<T> = Promise<{ data: { success: boolean; message?: string; data: T; meta?: Meta } }>;
interface Meta { total: number; page: number; limit: number; pages: number; unreadCount?: number }

// ── Auth ─────────────────────────────────────────────────────

export const authApi = {
  register: (body: { name: string; email: string; password: string; role: string }) =>
    api.post<{ success: boolean; data: { user: User; accessToken: string } }>("/auth/register", body),
  login: (email: string, password: string) =>
    api.post<{ success: boolean; data: { user: User; accessToken: string } }>("/auth/login", { email, password }),
  logout: () => api.post("/auth/logout"),
  me: () => api.get<{ success: boolean; data: User }>("/auth/me"),
  verifyEmail: (token: string) => api.get(`/auth/verify-email/${token}`),
  forgotPassword: (email: string) => api.post("/auth/forgot-password", { email }),
  resetPassword: (token: string, password: string) =>
    api.post(`/auth/reset-password/${token}`, { password }),
};

// ── Users ────────────────────────────────────────────────────

export const usersApi = {
  getMe: () => api.get<{ success: boolean; data: { user: User; socialAccounts: SocialAccount[] } }>("/users/me"),
  updateProfile: (body: { name?: string; avatar?: string }) => api.patch("/users/me", body),
  getInfluenceScore: () => api.get("/users/influence-score"),
  connectSocialAccount: (body: object) => api.post<{ success: boolean; data: SocialAccount }>("/users/social-accounts", body),
  deleteSocialAccount: (id: string) => api.delete(`/users/social-accounts/${id}`),
};

// ── Campaigns ────────────────────────────────────────────────

export const campaignsApi = {
  list: (params?: object) => api.get<{ success: boolean; data: Campaign[]; meta: Meta }>("/campaigns", { params }),
  getMy: (params?: object) => api.get<{ success: boolean; data: Campaign[]; meta: Meta }>("/campaigns/my", { params }),
  get: (id: string) => api.get<{ success: boolean; data: Campaign }>(`/campaigns/${id}`),
  create: (body: object) => api.post<{ success: boolean; data: Campaign }>("/campaigns", body),
  update: (id: string, body: object) => api.patch<{ success: boolean; data: Campaign }>(`/campaigns/${id}`, body),
  delete: (id: string) => api.delete(`/campaigns/${id}`),
};

// ── Submissions ──────────────────────────────────────────────

export const submissionsApi = {
  list: (params?: object) =>
    api.get<{ success: boolean; data: CampaignSubmission[]; meta: Meta }>("/submissions", { params }),
  get: (id: string) => api.get<{ success: boolean; data: CampaignSubmission }>(`/submissions/${id}`),
  create: (body: { campaignId: string; socialAccountId: string; repostUrl: string; screenshotUrl: string }) =>
    api.post<{ success: boolean; data: CampaignSubmission }>("/submissions", body),
  uploadScreenshot: (file: File) => {
    const fd = new FormData();
    fd.append("screenshot", file);
    return api.post<{ success: boolean; data: { url: string } }>("/submissions/upload", fd, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
  approve: (id: string) => api.post(`/submissions/${id}/approve`),
  reject: (id: string, reason: string) => api.post(`/submissions/${id}/reject`, { reason }),
};

// ── Wallet ───────────────────────────────────────────────────

export const walletApi = {
  get: () => api.get<{ success: boolean; data: { wallet: Wallet; transactions: Transaction[] } }>("/wallet"),
  getTransactions: (params?: object) =>
    api.get<{ success: boolean; data: Transaction[]; meta: Meta }>("/wallet/transactions", { params }),
  createTopup: (amount: number) =>
    api.post<{ success: boolean; data: { clientSecret: string; paymentIntentId: string; amount: number } }>("/wallet/topup", { amount }),
  withdraw: (amount: number) =>
    api.post<{ success: boolean; data: Withdrawal }>("/wallet/withdraw", { amount }),
  getWithdrawals: (params?: object) =>
    api.get<{ success: boolean; data: Withdrawal[]; meta: Meta }>("/wallet/withdrawals", { params }),
  createConnect: () =>
    api.post<{ success: boolean; data: { url: string; connectAccountId: string } }>("/wallet/connect"),
  getConnectStatus: () => api.get("/wallet/connect/status"),
};

// ── Admin ────────────────────────────────────────────────────

export const adminApi = {
  getStats: () => api.get("/admin/stats"),
  listUsers: (params?: object) => api.get<{ success: boolean; data: User[]; meta: Meta }>("/admin/users", { params }),
  getUser: (id: string) => api.get<{ success: boolean; data: User }>(`/admin/users/${id}`),
  suspendUser: (id: string, reason?: string) => api.post(`/admin/users/${id}/suspend`, { reason }),
  unsuspendUser: (id: string) => api.post(`/admin/users/${id}/unsuspend`),
  listFraudFlags: (params?: object) => api.get("/admin/fraud-flags", { params }),
  resolveFraudFlag: (id: string) => api.post(`/admin/fraud-flags/${id}/resolve`),
  listWithdrawals: (params?: object) =>
    api.get<{ success: boolean; data: Withdrawal[]; meta: Meta }>("/admin/withdrawals", { params }),
  approveWithdrawal: (id: string) => api.post(`/admin/withdrawals/${id}/approve`),
  rejectWithdrawal: (id: string, reason?: string) => api.post(`/admin/withdrawals/${id}/reject`, { reason }),
  listSubmissions: (params?: object) =>
    api.get<{ success: boolean; data: CampaignSubmission[]; meta: Meta }>("/submissions", { params }),
};

// ── Notifications ────────────────────────────────────────────

export const notificationsApi = {
  list: (params?: object) =>
    api.get<{ success: boolean; data: Notification[]; meta: Meta & { unreadCount: number } }>("/notifications", { params }),
  markRead: (id: string) => api.post(`/notifications/${id}/read`),
  markAllRead: () => api.post("/notifications/read-all"),
};
