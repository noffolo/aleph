import { createConnectTransport } from "@connectrpc/connect-web";
import type { Interceptor } from "@connectrpc/connect";

const STORAGE_KEY = "aleph_api_key";

export const getStoredApiKey = () => localStorage.getItem(STORAGE_KEY) || "";

export const setApiKey = (key: string) => {
  localStorage.setItem(STORAGE_KEY, key);
};

export const clearApiKey = () => {
  localStorage.removeItem(STORAGE_KEY);
};

const authInterceptor: Interceptor = (next) => async (req) => {
  const key = getStoredApiKey();
  if (key) {
    req.header.set("X-Aleph-Api-Key", key);
  }
  return await next(req);
};

export const transport = createConnectTransport({
  baseUrl: "http://localhost:8080",
  interceptors: [authInterceptor],
});
