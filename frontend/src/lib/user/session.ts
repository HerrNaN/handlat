import { get, writable } from "svelte/store";
import { queryClient } from "../query";

export type Account = {
  id: string;
  name: string;
};

/** `undefined` until the first check settles, then the account or `null`. */
const current = writable<Account | null | undefined>(undefined);

export const session = { subscribe: current.subscribe };

export const clearSession = () => {
  queryClient.clear();
  current.set(null);
};

export const hasActiveSession = async (): Promise<boolean> => {
  if (get(current) === undefined) {
    await refreshSession();
  }

  return get(current) !== null && get(current) !== undefined;
};

export const refreshSession = async () => {
  try {
    const account = await queryClient.fetchQuery({
      queryKey: ["session"],
      queryFn: async ({ signal }) => {
        const response = await fetch("/api/auth/me", { cache: "no-store", signal });
        if (response.status === 401) return null;
        if (!response.ok) throw new Error("Could not load session");
        return await response.json() as Account;
      },
    });
    if (account) current.set(account);
    else clearSession();
  } catch {
    // Server unreachable: keep what we already know instead of signing out.
  }
};
