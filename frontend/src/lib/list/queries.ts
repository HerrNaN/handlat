import type { MutationOptions, QueryFunctionContext } from "@tanstack/query-core";
import { get } from "svelte/store";
import { queryClient } from "../query";
import { session } from "../user/session";
import { addItem, clearCheckedItems, createList, deleteItem, getList, getLists, renameList, setItemChecked, updateItem, type ItemChanges, type NewItem, type ShoppingList } from "./api";

type Owner = { account: string };
type ListTarget = Owner & { listID: string };
type ItemTarget = ListTarget & { itemID: string };
type CheckItem = ItemTarget & { checked: boolean };
type OptimisticCheck = { previous: Date | undefined; optimistic: Date | undefined };

export const summariesKey = (account?: string) => ["private-lists", account, "summaries"] as const;
export const listKey = (account: string | undefined, id: string) => ["private-lists", account, "detail", id] as const;

const refetchOnWindowFocus = () => queryClient.isMutating({ mutationKey: ["private-lists"] }) === 0;

export const summariesQuery = (account?: string) => ({
  queryKey: summariesKey(account),
  enabled: Boolean(account),
  refetchOnWindowFocus,
  refetchOnMount: refetchOnWindowFocus,
  queryFn: ({ signal }: QueryFunctionContext) => getLists(signal),
});

export const listQuery = (account: string | undefined, id: string) => ({
  queryKey: listKey(account, id),
  enabled: Boolean(account && id),
  refetchOnWindowFocus,
  refetchOnMount: refetchOnWindowFocus,
  queryFn: ({ signal }: QueryFunctionContext) => getList(id, signal),
});

export const dismissListErrors = () => {
  const cache = queryClient.getMutationCache();
  cache.findAll({ mutationKey: ["private-lists"], status: "error" }).forEach((mutation) => cache.remove(mutation));
};

const listMutationOptions = {
  mutationKey: ["private-lists", "write"],
  onMutate: async ({ account }: Owner) => {
    dismissListErrors();
    await queryClient.cancelQueries({ queryKey: ["private-lists"] });
    if (account !== get(session)?.id) throw new Error("The active account changed.");
  },
  onSettled: (_data: ShoppingList | undefined, _error: Error | null, { account }: Owner) => {
    if (account !== get(session)?.id) return;
    // This mutation still counts as pending until onSettled completes.
    if (queryClient.isMutating({ mutationKey: ["private-lists"] }) === 1) {
      return queryClient.invalidateQueries({ queryKey: ["private-lists", account] });
    }
  },
};

export const createListMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ name }: Owner & { name: string }) => createList(name),
});

export const renameListMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, name }: ListTarget & { name: string }) => renameList(listID, name),
});

export const addItemMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, item }: ListTarget & { item: NewItem }) => addItem(listID, item),
});

export const updateItemMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, itemID, changes }: ItemTarget & { changes: ItemChanges }) => updateItem(listID, itemID, changes),
});

export const deleteItemMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, itemID }: ItemTarget) => deleteItem(listID, itemID),
});

export const clearCheckedMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID }: ListTarget) => clearCheckedItems(listID),
});

export const checkItemMutation = (): MutationOptions<ShoppingList, Error, CheckItem, OptimisticCheck | undefined> => ({
  ...listMutationOptions,
  mutationFn: ({ listID, itemID, checked }) => setItemChecked(listID, itemID, checked),
  onMutate: async (target) => {
    await listMutationOptions.onMutate(target);
    const key = listKey(target.account, target.listID);
    const previous = queryClient.getQueryData<ShoppingList>(key)?.items.find((item) => item.id === target.itemID);
    if (!previous) return;
    const optimistic = target.checked ? new Date() : undefined;
    queryClient.setQueryData<ShoppingList>(key, (list) => list && ({
      ...list,
      items: list.items.map((item) => item.id === target.itemID ? { ...item, checkedAt: optimistic } : item),
    }));
    return { previous: previous.checkedAt, optimistic };
  },
  onError: (_error, target, snapshot) => {
    if (target.account !== get(session)?.id || !snapshot) return;
    const key = listKey(target.account, target.listID);
    queryClient.setQueryData<ShoppingList>(key, (list) => list && ({
      ...list,
      items: list.items.map((item) => item.id === target.itemID && item.checkedAt === snapshot.optimistic
        ? { ...item, checkedAt: snapshot.previous }
        : item),
    }));
  },
});
