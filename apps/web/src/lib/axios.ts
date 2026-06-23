import axios from "axios";

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api",
  withCredentials: true, // sends httpOnly refresh token cookie
  headers: { "Content-Type": "application/json" },
});

// Attach access token from memory (set by auth store)
api.interceptors.request.use((config) => {
  const token =
    typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Auto-refresh on 401
api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;
      try {
        const { data } = await axios.post(
          `${process.env.NEXT_PUBLIC_API_URL}/auth/refresh`,
          {},
          { withCredentials: true }
        );
        localStorage.setItem("access_token", data.data.accessToken);
        original.headers.Authorization = `Bearer ${data.data.accessToken}`;
        return api(original);
      } catch {
        localStorage.removeItem("access_token");

        // Cookie refresh failed — try localStorage refresh token (PWA fallback).
        const storedRT = localStorage.getItem("refresh_token");
        if (storedRT) {
          try {
            const { data: fb } = await axios.post(
              `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api"}/auth/refresh`,
              { refreshToken: storedRT },
              { withCredentials: true }
            );
            const newToken = fb.data.accessToken as string;
            localStorage.setItem("access_token", newToken);
            original.headers.Authorization = `Bearer ${newToken}`;
            return api(original);
          } catch {
            localStorage.removeItem("refresh_token");
          }
        }

        window.location.href = "/login";
        return Promise.reject(error);
      }
    }
    return Promise.reject(error);
  }
);

export default api;
