import { QueryObserver } from "@tanstack/query-core";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { queryClient } from "../query";
import { clearSession, refreshSession } from "../user/session";
import { type ShoppingList } from "./api";
import {
  checkItemMutation,
  deleteListMutation,
  listKey,
  listQuery,
  summariesKey,
  summariesQuery,
} from "./queries";

class MockEventSource extends EventTarget {
  static CLOSED = 2;
  static instances: MockEventSource[] = [];
  readyState = 1;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public url: string) {
    super();
    MockEventSource.instances.push(this);
  }
  close = vi.fn(() => {
    this.readyState = MockEventSource.CLOSED;
  });
  snapshot = (value: unknown) =>
    this.onmessage?.(
      new MessageEvent("message", {
        data: JSON.stringify(value),
      }),
    );
}

const snapshot = (name = "Groceries"): ShoppingList => ({
  id: "list",
  name,
  items: [{ id: "milk", text: "Milk", createdAt: new Date("2026-11-08T12:00:00Z") }],
});
const wire = (list: ShoppingList) => JSON.parse(JSON.stringify(list));
const fetch = vi.fn<typeof globalThis.fetch>();
let stop: (() => void) | undefined;
const observe = async (deleted = vi.fn()) => {
  const count = MockEventSource.instances.length;
  const observer = new QueryObserver(queryClient, listQuery("owner", "list", deleted));
  stop = observer.subscribe(() => {});
  await vi.waitFor(() => expect(MockEventSource.instances).toHaveLength(count + 1));
  return observer;
};
const source = () => MockEventSource.instances.at(-1)!;
const data = () => queryClient.getQueryData<ShoppingList>(listKey("owner", "list"));

beforeEach(async () => {
  MockEventSource.instances = [];
  fetch.mockReset();
  vi.stubGlobal("fetch", fetch);
  vi.stubGlobal("EventSource", MockEventSource);
  clearSession();
  fetch.mockResolvedValueOnce(new Response(JSON.stringify({ id: "owner", name: "Owner" })));
  await refreshSession();
});
afterEach(() => {
  stop?.();
  stop = undefined;
  clearSession();
  vi.unstubAllGlobals();
});

it("keeps the stream alive through an optimistic edit and ignores the HTTP snapshot", async () => {
  const observer = await observe();
  expect(observer.getCurrentResult().isPending).toBe(true);
  source().snapshot(wire(snapshot()));
  await vi.waitFor(() => expect(data()?.name).toBe("Groceries"));
  let resolve!: (response: Response) => void;
  fetch.mockImplementationOnce(
    () =>
      new Promise((done) => {
        resolve = done;
      }),
  );
  const mutation = queryClient.getMutationCache().build(queryClient, checkItemMutation());
  const saving = mutation.execute({
    account: "owner",
    listID: "list",
    itemID: "milk",
    checked: true,
  });
  await vi.waitFor(() => expect(data()?.items[0]?.checkedAt).toBeInstanceOf(Date));
  expect(source().close).not.toHaveBeenCalled();
  const confirmed = snapshot("From SSE");
  confirmed.items[0]!.checkedAt = new Date("2026-11-08T12:01:00Z");
  source().snapshot(wire(confirmed));
  await vi.waitFor(() => expect(data()?.name).toBe("From SSE"));
  resolve(new Response(JSON.stringify(wire(snapshot("Older HTTP response")))));
  await saving;
  expect(data()).toEqual(confirmed);
  expect(MockEventSource.instances).toHaveLength(1);
  expect(observer.getCurrentResult().isFetching).toBe(true);
});

it("closes on refresh and unmount, and opens a new stream on return", async () => {
  const observer = await observe();
  source().snapshot(wire(snapshot()));
  await vi.waitFor(() => expect(data()).toBeDefined());
  const first = source();
  void observer.refetch();
  await vi.waitFor(() => expect(MockEventSource.instances).toHaveLength(2));
  expect(first.close).toHaveBeenCalled();
  source().snapshot(wire(snapshot("Refreshed")));
  await vi.waitFor(() => expect(data()?.name).toBe("Refreshed"));
  const second = source();
  stop?.();
  expect(second.close).toHaveBeenCalled();
  await observe();
  await vi.waitFor(() => expect(MockEventSource.instances).toHaveLength(3));
  source().snapshot(wire(snapshot("Returned")));
  await vi.waitFor(() => expect(data()?.name).toBe("Returned"));
});

it("lets EventSource reconnect transient failures and exposes terminal failures for retry", async () => {
  const observer = await observe();
  const first = source();
  first.readyState = 0;
  first.onerror?.();
  expect(first.close).not.toHaveBeenCalled();
  first.readyState = MockEventSource.CLOSED;
  first.onerror?.();
  await vi.waitFor(() => expect(observer.getCurrentResult().isError).toBe(true));
  void observer.refetch();
  await vi.waitFor(() => expect(MockEventSource.instances).toHaveLength(2));
  source().snapshot(wire(snapshot()));
  await vi.waitFor(() => expect(observer.getCurrentResult().isSuccess).toBe(true));
});

it("closes an idle stream on sign out and rejects late snapshots", async () => {
  await observe();
  const first = source();
  clearSession();
  expect(first.close).toHaveBeenCalled();
  first.snapshot(wire(snapshot()));
  await Promise.resolve();
  expect(data()).toBeUndefined();
});

it("removes a deleted list and notifies the page", async () => {
  const deleted = vi.fn();
  await observe(deleted);
  source().snapshot(wire(snapshot()));
  await vi.waitFor(() => expect(data()).toBeDefined());
  source().dispatchEvent(new Event("deleted"));
  expect(deleted).toHaveBeenCalledOnce();
  expect(source().close).toHaveBeenCalled();
  await Promise.resolve();
  expect(data()).toBeUndefined();
});

it("cancels an older overview fetch when deleting a list", async () => {
  const key = summariesKey("owner");
  const lists = [{ id: "list", name: "Groceries", itemCount: 1 }];
  queryClient.setQueryData(key, lists);
  let resolve!: (response: Response) => void;
  let signal: AbortSignal | undefined;
  fetch.mockImplementationOnce((_url, init) => {
    signal = init?.signal ?? undefined;
    return new Promise((done) => {
      resolve = done;
    });
  });
  const observer = new QueryObserver(queryClient, summariesQuery("owner"));
  stop = observer.subscribe(() => {});
  expect(signal?.aborted).toBe(false);
  fetch.mockResolvedValueOnce(new Response(null, { status: 204 }));
  fetch.mockResolvedValueOnce(new Response("[]"));
  const mutation = queryClient.getMutationCache().build(queryClient, deleteListMutation());
  await mutation.execute({ account: "owner", listID: "list" });
  expect(signal?.aborted).toBe(true);
  resolve(new Response(JSON.stringify(lists)));
  await vi.waitFor(() => expect(queryClient.getQueryData(key)).toEqual([]));
});
