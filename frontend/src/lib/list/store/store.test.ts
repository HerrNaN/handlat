import { describe, it, expect, vi } from "vitest";
import { get } from "svelte/store";
import { createItemsStore } from "./store";

const initial = { id: "list-1", name: "Groceries", items: [] };
const milk = { id: "milk", text: "Milk", createdAt: "2026-10-23T10:00:00Z", department: "other" };
const response = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status });
const setup = () => {
  const fetch = vi.fn<typeof globalThis.fetch>();
  const onUnauthorized = vi.fn();
  const store = createItemsStore({ fetch, onUnauthorized });
  store.setAccount("owner");
  return { store, fetch, onUnauthorized };
};

describe("API list store", () => {
  it("does not fetch or create defaults before an account is available", async () => {
    const fetch = vi.fn<typeof globalThis.fetch>();
    const store = createItemsStore({ fetch });
    expect(await store.refreshLists()).toBe(false);
    expect(fetch).not.toHaveBeenCalled();
    expect(get(store.lists)).toEqual([]);
  });

  it("loads summaries and opens a list directly by URL", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response([{ ...initial, itemCount: 0 }]));
    expect(await store.refreshLists()).toBe(true);
    expect(get(store.lists)[0]?.name).toBe("Groceries");
    fetch.mockResolvedValueOnce(response({ ...initial, items: [milk] }));
    expect(await store.setActiveList(initial.id)).toBe(true);
    expect(get(store.activeList).items[0]?.createdAt).toBeInstanceOf(Date);
    expect(fetch).toHaveBeenLastCalledWith("/api/lists/list-1", { cache: "no-store" });
  });

  it("creates lists only when requested", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response(initial, 201));
    expect(await store.createList("Groceries")).toBe(true);
    expect(get(store.activeList).id).toBe(initial.id);
    expect(get(store.lists)).toHaveLength(1);
    expect(fetch.mock.calls[0]?.[1]?.body).toBe(JSON.stringify({ name: "Groceries" }));
  });

  it("only applies saves after acknowledgement and does not queue overlapping edits", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response(initial));
    await store.setActiveList(initial.id);
    let resolve!: (value: Response) => void;
    fetch.mockImplementationOnce(() => new Promise((done) => { resolve = done; }));
    const saving = store.addItem("Milk");
    expect(get(store.status).saving).toBe(true);
    expect(get(store.activeList).items).toEqual([]);
    expect(await store.addItem("Bread")).toBe(false);
    resolve(response({ ...initial, items: [milk] }));
    expect(await saving).toBe(true);
    expect(get(store.activeList).items).toHaveLength(1);
    expect(get(store.lists)[0]?.itemCount).toBe(1);
    expect(fetch.mock.calls[1]?.[1]?.headers).toEqual({ "Content-Type": "application/json" });
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("sends independent fields without stale item data or write preconditions", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response({ ...initial, items: [milk] }));
    await store.setActiveList(initial.id);
    fetch.mockResolvedValueOnce(response({ ...initial, items: [{ ...milk, text: "Oat milk", department: "dairy", checkedAt: "2026-10-23T11:00:00Z" }] }));
    expect(await store.updateItem("milk", { text: "Oat milk" })).toBe(true);
    expect(JSON.parse(fetch.mock.calls[1]?.[1]?.body as string)).toEqual({ text: "Oat milk" });
    expect(fetch.mock.calls[1]?.[1]?.method).toBe("PATCH");
    expect(get(store.activeList).items[0]?.department).toBe("dairy");
    expect(get(store.activeList).items[0]?.checkedAt).toBeInstanceOf(Date);
    expect(get(store.status).error).toBe("");
  });

  it("reports offline saves without applying or queuing them", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response(initial));
    await store.setActiveList(initial.id);
    fetch.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    expect(await store.addItem("Milk")).toBe(false);
    expect(get(store.status).error).toContain("internet connection");
    expect(get(store.activeList).items).toEqual([]);
    expect(get(store.status).saving).toBe(false);
  });

  it("clears data and ignores pending responses on account changes", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response(initial));
    await store.setActiveList(initial.id);
    let resolve!: (value: Response) => void;
    fetch.mockImplementationOnce(() => new Promise((done) => { resolve = done; }));
    const pending = store.addItem("Milk");
    store.setAccount("other");
    resolve(response({ ...initial, items: [milk] }));
    expect(await pending).toBe(false);
    expect(get(store.lists)).toEqual([]);
    expect(get(store.activeList).id).toBe("");
  });

  it("invalidates the session when the API rejects it", async () => {
    const { store, fetch, onUnauthorized } = setup();
    fetch.mockResolvedValueOnce(response({}, 401));
    expect(await store.refreshLists()).toBe(false);
    expect(onUnauthorized).toHaveBeenCalledOnce();
    expect(get(store.status).error).toContain("sign in again");
  });

  it("sends explicit checked states, not toggles or browser timestamps", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response({ ...initial, items: [milk] }));
    await store.setActiveList(initial.id);
    fetch.mockResolvedValueOnce(response({ ...initial, items: [{ ...milk, checkedAt: "2026-10-23T11:00:00Z" }] }));
    expect(await store.setItemChecked("milk", true)).toBe(true);
    expect(JSON.parse(fetch.mock.calls[1]?.[1]?.body as string)).toEqual({ checked: true });
    fetch.mockResolvedValueOnce(response({ ...initial, items: [milk] }));
    await store.setItemChecked("milk", false);
    expect(JSON.parse(fetch.mock.calls[2]?.[1]?.body as string)).toEqual({ checked: false });
    fetch.mockResolvedValueOnce(response({ ...initial, items: [] }));
    expect(await store.deleteItem("milk")).toBe(true);
    expect(fetch.mock.calls[3]?.[1]?.method).toBe("DELETE");
    expect(get(store.activeList).items).toEqual([]);
  });

  it("reuses an addition ID after a lost response, but not for a distinct addition", async () => {
    const { store, fetch } = setup();
    fetch.mockResolvedValueOnce(response(initial));
    await store.setActiveList(initial.id);
    fetch.mockRejectedValueOnce(new TypeError("Response lost after commit"));
    expect(await store.addItem("Milk")).toBe(false);
    const first = JSON.parse(fetch.mock.calls[1]?.[1]?.body as string);
    expect(first.id).toBeTruthy();
    // Refreshing must not discard the identity of an unacknowledged addition.
    fetch.mockResolvedValueOnce(response({ ...initial, items: [{ ...milk, id: first.id }] }));
    await store.refresh();
    fetch.mockResolvedValueOnce(response({ ...initial, items: [{ ...milk, id: first.id }] }));
    expect(await store.addItem("Milk")).toBe(true);
    expect(JSON.parse(fetch.mock.calls[3]?.[1]?.body as string)).toEqual(first);
    fetch.mockResolvedValueOnce(response({ ...initial, items: [{ ...milk, id: first.id }, milk] }));
    expect(await store.addItem("Milk")).toBe(true);
    expect(JSON.parse(fetch.mock.calls[4]?.[1]?.body as string).id).not.toBe(first.id);
  });
});
