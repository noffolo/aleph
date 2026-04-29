import { createConnectTransport } from "@connectrpc/connect-web";

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

export const transport = createConnectTransport({
  baseUrl: "",
  credentials: "include",
});
