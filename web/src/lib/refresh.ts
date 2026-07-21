// Decodes a JWT payload (base64url) and checks whether it is expired.
export function isJwtExpired(token: string): boolean {
  try {
    const b64 = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const payload = JSON.parse(atob(b64));
    return Date.now() / 1000 >= payload.exp;
  } catch {
    return true; // treat undecoded tokens as expired
  }
}

// Tries to get a fresh access token.
// Attempt 1: cookie-based refresh (normal flow).
// Attempt 2: localStorage refresh token (PWA fallback — OS may clear cookies after
//            long background / aggressive battery optimisation on Android OEMs).
// Returns the new access token, or null if both attempts fail.
export async function attemptRefresh(): Promise<string | null> {
  const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api";

  // Attempt 1 — cookie
  const cookieRes = await fetch(`${base}/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
  }).catch(() => null);

  if (cookieRes?.ok) {
    const body = await cookieRes.json();
    return body.data.accessToken as string;
  }

  // Attempt 2 — localStorage token in request body
  const storedRT = typeof window !== "undefined" ? localStorage.getItem("refresh_token") : null;
  if (!storedRT) return null;

  const bodyRes = await fetch(`${base}/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken: storedRT }),
  }).catch(() => null);

  if (bodyRes?.ok) {
    const body = await bodyRes.json();
    return body.data.accessToken as string;
  }

  // Token is definitively invalid — clean it up
  if (bodyRes?.status === 401) {
    localStorage.removeItem("refresh_token");
  }

  return null;
}
