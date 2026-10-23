import { get, writable } from "svelte/store";
import { ItemSchema, type Item } from "../list";
import { classify, departments } from "../classifier";
import z from "zod";

export type ListSummary = { id: string; name: string; itemCount: number };
export type ActiveList = { id: string; name: string; items: Item[] };
export type ListStatus = { loading: boolean; saving: boolean; error: string };

const ListSchema = z.object({
  id: z.string(), name: z.string(),
  items: z.array(ItemSchema),
});
const SummariesSchema = z.array(z.object({
  id: z.string(), name: z.string(), itemCount: z.number().int(),
}));
const emptyList = (): ActiveList => ({ id: "", name: "", items: [] });

class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

export const createItemsStore = (deps: {
  fetch: typeof fetch;
  onUnauthorized?: () => void;
  randomUUID?: () => string;
}) => {
  const lists = writable<ListSummary[]>([]);
  const activeList = writable<ActiveList>(emptyList());
  const status = writable<ListStatus>({ loading: false, saving: false, error: "" });
  let generation = 0;
  let accountID: string | undefined;
  let selectedID = "";
  // A failed response may hide a committed addition. Keep its identity in memory
  // for explicit retries; never queue or replay it automatically.
  const pendingAdditions = new Map<string, string>();

  const reset = (id?: string) => {
    generation++;
    accountID = id;
    selectedID = "";
    pendingAdditions.clear();
    lists.set([]);
    activeList.set(emptyList());
    status.set({ loading: false, saving: false, error: "" });
  };
  const setAccount = (id?: string) => {
    if (id !== accountID) reset(id);
  };

  const request = async (path: string, init?: RequestInit) => {
    const response = await deps.fetch(`/api/lists${path}`, { cache: "no-store", ...init });
    if (!response.ok) {
      throw new ApiError(response.status,
        response.status === 401 ? "Your session expired. Please sign in again."
        : response.status === 404 ? "This list or item is no longer available."
        : response.status >= 500 ? "The server could not save or load your list. Try again."
        : (await response.text()) || "The request failed.");
    }
    return response.json();
  };

  const run = async (saving: boolean, work: (version: number) => Promise<void>) => {
    if (!accountID || get(status).loading || get(status).saving) return false;
    const version = generation;
    status.set({ loading: !saving, saving, error: "" });
    try {
      await work(version);
      return version === generation;
    } catch (error) {
      if (version !== generation) return false;
      if (error instanceof ApiError && error.status === 401) deps.onUnauthorized?.();
      status.set({ loading: false, saving: false, error: error instanceof ApiError ? error.message : "Unable to reach the server. Lists require an internet connection. Refresh to check server state before retrying your change." });
      return false;
    } finally {
      if (version === generation) status.update((value) => ({ ...value, loading: false, saving: false }));
    }
  };

  const apply = (list: ActiveList) => {
    activeList.set(list);
    lists.update((values) => {
      const summary = { id: list.id, name: list.name, itemCount: list.items.length };
      return values.some((value) => value.id === list.id)
        ? values.map((value) => value.id === list.id ? summary : value)
        : [...values, summary];
    });
  };

  const refreshLists = () => {
    generation++;
    selectedID = "";
    activeList.set(emptyList());
    status.set({ loading: false, saving: false, error: "" });
    return run(false, async (version) => {
      const values = SummariesSchema.parse(await request(""));
      if (version === generation) lists.set(values);
    });
  };

  const setActiveList = (id: string) => {
    generation++;
    selectedID = id;
    activeList.set(emptyList());
    status.set({ loading: false, saving: false, error: "" });
    return run(false, async (version) => {
      const value = ListSchema.parse(await request(`/${encodeURIComponent(id)}`));
      if (version === generation) apply(value);
    });
  };

  const refresh = () => {
    if (get(status).saving || get(status).loading) return Promise.resolve(false);
    return selectedID ? setActiveList(selectedID) : refreshLists();
  };

  const mutate = (method: string, suffix: string, body?: unknown) => run(true, async (version) => {
    const list = get(activeList);
    if (!list.id) throw new ApiError(404, "Open a list before editing.");
    const value = ListSchema.parse(await request(`/${encodeURIComponent(list.id)}${suffix}`, {
      method,
      headers: { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    }));
    if (version === generation) apply(value);
  });

  const updateItem = (id: string, updates: Partial<Pick<Item, "text" | "department">>) =>
    mutate("PATCH", `/items/${encodeURIComponent(id)}`, updates);

  const addItem = async (text: string) => {
    if (!accountID || get(status).loading || get(status).saving) return false;
    const key = JSON.stringify([get(activeList).id, text]);
    const id = pendingAdditions.get(key) ?? (deps.randomUUID ?? (() => crypto.randomUUID()))();
    pendingAdditions.set(key, id);
    const saved = await mutate("POST", "/items", { id, text, department: classify(text, departments).id });
    if (saved) pendingAdditions.delete(key);
    return saved;
  };

  return {
    lists: { subscribe: lists.subscribe },
    activeList: { subscribe: activeList.subscribe },
    status: { subscribe: status.subscribe },
    setAccount, reset, refreshLists, setActiveList, refresh,
    createList: (name = "Shopping List") => run(true, async (version) => {
      const list = ListSchema.parse(await request("", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name }) }));
      if (version === generation) apply(list);
    }),
    changeListName: (name: string) => mutate("PATCH", "", { name }),
    addItem,
    updateItem,
    setItemChecked: (id: string, checked: boolean) => mutate("PATCH", `/items/${encodeURIComponent(id)}`, { checked }),
    deleteItem: (id: string) => mutate("DELETE", `/items/${encodeURIComponent(id)}`),
    deleteCheckedItems: () => mutate("DELETE", "/checked-items"),
  };
};
