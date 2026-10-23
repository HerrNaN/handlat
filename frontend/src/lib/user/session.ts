import { get, writable } from "svelte/store";

export type Account = {
  id: string;
  name: string;
};

/** `undefined` until the first check settles, then the account or `null`. */
const current = writable<Account | null | undefined>(undefined);

export const session = { subscribe: current.subscribe };

export const clearSession = () => current.set(null);

export const hasActiveSession = async (): Promise<boolean> => {
  if (get(current) === undefined) {
    await refreshSession();
  }

  return get(current) !== null && get(current) !== undefined;
};

export const refreshSession = async () => {
  try {
    const response = await fetch("/api/auth/me", { cache: "no-store" });
    if (response.ok) {
      current.set((await response.json()) as Account);
    } else if (response.status === 401) {
      current.set(null);
    }
  } catch {
    // Server unreachable: keep what we already know instead of signing out.
  }
};
