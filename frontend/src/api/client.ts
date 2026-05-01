import { createConnectTransport } from "@connectrpc/connect-web";
import { useStore } from "../store/useStore";

export const createSession = async (apiKey: string) => {
  const res = await fetch("/api/v1/auth/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ api_key: apiKey }),
    credentials: "include",
  });
  if (!res.ok) throw new Error("Invalid API key");
  return res.json();
};

export const deleteSession = async () => {
  await fetch("/api/v1/auth/session", {
    method: "DELETE",
    credentials: "include",
  });
};

export const apiGet = async (path: string) => {
  const state = useStore.getState();
  const res = await fetch(path, {
    headers: {
      ...(state.apiKey ? { "Authorization": `Bearer ${state.apiKey}` } : {}),
    },
    credentials: "include",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || `API error: ${res.status}`);
  }
  return res.json();
};

export const apiPost = async (path: string, body: unknown) => {
  const state = useStore.getState();
  const res = await fetch(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(state.apiKey ? { "Authorization": `Bearer ${state.apiKey}` } : {}),
    },
    body: JSON.stringify(body),
    credentials: "include",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || `API error: ${res.status}`);
  }
  return res.json();
};

export const apiPatch = (path: string, body: unknown) => apiPost(path, body);

export const transport = createConnectTransport({
  baseUrl: "",
  credentials: "include",
});
