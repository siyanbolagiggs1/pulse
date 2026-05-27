"use client";
import { create } from "zustand";
import type { User } from "@/types";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  setAuth: (user: User, token: string) => void;
  updateUser: (user: User) => void;
  clearAuth: () => void;
  setLoading: (v: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,
  setAuth: (user, token) => {
    if (typeof window !== "undefined") {
      sessionStorage.setItem("access_token", token);
    }
    set({ user, isLoading: false });
  },
  updateUser: (user) => set({ user }),
  clearAuth: () => {
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("access_token");
    }
    set({ user: null, isLoading: false });
  },
  setLoading: (isLoading) => set({ isLoading }),
}));
