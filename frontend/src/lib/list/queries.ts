import {
  experimental_streamedQuery as streamedQuery,
  type MutationOptions,
  type QueryFunctionContext,
} from "@tanstack/query-core";
import { get } from "svelte/store";

import { queryClient } from "../query";
import { session } from "../user/session";
import {
  addItem,
  clearCheckedItems,
  createList,
  deleteItem,
  leaveList,
  getLists,
  renameList,
  setItemChecked,
  updateItem,
  type ItemChanges,
  type ListSummary,
  type NewItem,
  type ShoppingList,
} from "./api";
import { streamList } from "./stream";

type Account = { account: string };
type ListTarget = Account & { listID: string };
type ItemTarget = ListTarget & { itemID: string };
type CheckItem = ItemTarget & { checked: boolean };
type OptimisticCheck = { previous: Date | undefined; optimistic: Date | undefined };

export const summariesKey = (account?: string) => ["private-lists", account, "summaries"] as const;
export const listKey = (account: string | undefined, id: string) =>
  ["private-lists", account, "detail", id] as const;

const refetchOnWindowFocus = () => queryClient.isMutating({ mutationKey: ["private-lists"] }) === 0;

export const summariesQuery = (account?: string) => ({
  queryKey: summariesKey(account),
  enabled: Boolean(account),
  refetchOnWindowFocus,
  refetchOnMount: refetchOnWindowFocus,
  queryFn: ({ signal }: QueryFunctionContext) => getLists(signal),
});

export const listQuery = (account: string | undefined, id: string, onDeleted: () => void) => ({
  queryKey: listKey(account, id),
  enabled: Boolean(account && id),
  refetchOnMount: "always" as const,
  queryFn: streamedQuery({
    streamFn: ({ signal }) => ({
      async *[Symbol.asyncIterator]() {
        const deleted = () => {
          queryClient.removeQueries({ queryKey: listKey(account, id), exact: true });
          void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
          onDeleted();
        };
        for await (const snapshot of streamList(id, signal, deleted)) {
          void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
          yield snapshot;
        }
      },
    }),
    // Each event is a complete list. Keep cached data until the next snapshot.
    refetchMode: "append",
    initialValue: undefined as ShoppingList | undefined,
    reducer: (_previous, snapshot: ShoppingList) => snapshot,
  }),
});

export const dismissListErrors = () => {
  const cache = queryClient.getMutationCache();
  cache
    .findAll({ mutationKey: ["private-lists"], status: "error" })
    .forEach((mutation) => cache.remove(mutation));
};

const listMutationOptions = {
  mutationKey: ["private-lists", "write"],
  onMutate: ({ account }: Account) => {
    dismissListErrors();
    if (account !== get(session)?.id) throw new Error("The active account changed.");
  },
  onSettled: (_data: ShoppingList | void, _error: Error | null, { account }: Account) => {
    if (account !== get(session)?.id) return;
    // This mutation still counts as pending until onSettled completes.
    if (queryClient.isMutating({ mutationKey: ["private-lists"] }) === 1) {
      return queryClient.invalidateQueries({ queryKey: summariesKey(account) });
    }
  },
};

export const createListMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ name }: Account & { name: string }) => createList(name),
});

export const renameListMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, name }: ListTarget & { name: string }) => renameList(listID, name),
});

export const leaveListMutation = () => ({
  ...listMutationOptions,
  onMutate: async (target: ListTarget) => {
    await queryClient.cancelQueries({ queryKey: summariesKey(target.account) });
    listMutationOptions.onMutate(target);
  },
  mutationFn: ({ listID }: ListTarget) => leaveList(listID),
  onSettled: (data: void, error: Error | null, target: ListTarget) => {
    // Leaving is complete when the server confirms it, not when refetching finishes.
    void listMutationOptions.onSettled(data, error, target);
  },
  onSuccess: (_data: void, { account, listID }: ListTarget) => {
    if (account !== get(session)?.id) return;
    queryClient.removeQueries({ queryKey: listKey(account, listID), exact: true });
    queryClient.setQueryData<ListSummary[]>(summariesKey(account), (lists) =>
      lists?.filter((list) => list.id !== listID),
    );
  },
});

export const addItemMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, item }: ListTarget & { item: NewItem }) => addItem(listID, item),
});

export const updateItemMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, itemID, changes }: ItemTarget & { changes: ItemChanges }) =>
    updateItem(listID, itemID, changes),
});

export const deleteItemMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID, itemID }: ItemTarget) => deleteItem(listID, itemID),
});

export const clearCheckedMutation = () => ({
  ...listMutationOptions,
  mutationFn: ({ listID }: ListTarget) => clearCheckedItems(listID),
});

export const checkItemMutation = (): MutationOptions<
  ShoppingList,
  Error,
  CheckItem,
  OptimisticCheck | undefined
> => ({
  ...listMutationOptions,
  mutationFn: ({ listID, itemID, checked }) => setItemChecked(listID, itemID, checked),
  onMutate: async (target) => {
    listMutationOptions.onMutate(target);
    const key = listKey(target.account, target.listID);
    const previous = queryClient
      .getQueryData<ShoppingList>(key)
      ?.items.find((item) => item.id === target.itemID);
    if (!previous) return;
    const optimistic = target.checked ? new Date() : undefined;
    queryClient.setQueryData<ShoppingList>(
      key,
      (list) =>
        list && {
          ...list,
          items: list.items.map((item) =>
            item.id === target.itemID ? { ...item, checkedAt: optimistic } : item,
          ),
        },
    );
    return { previous: previous.checkedAt, optimistic };
  },
  onError: (_error, target, snapshot) => {
    if (target.account !== get(session)?.id || !snapshot) return;
    const key = listKey(target.account, target.listID);
    queryClient.setQueryData<ShoppingList>(
      key,
      (list) =>
        list && {
          ...list,
          items: list.items.map((item) =>
            item.id === target.itemID && item.checkedAt === snapshot.optimistic
              ? { ...item, checkedAt: snapshot.previous }
              : item,
          ),
        },
    );
  },
});
