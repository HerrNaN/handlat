import { ListSchema, type ShoppingList } from "./api";

export const streamList = (listID: string, signal: AbortSignal, onDeleted: () => void) => ({
  async *[Symbol.asyncIterator]() {
    if (signal.aborted) return;
    const events = new EventSource(`/api/lists/${encodeURIComponent(listID)}/events`);
    let pending: ShoppingList | undefined;
    let error: unknown;
    let closed = false;
    let wake: (() => void) | undefined;
    const close = () => {
      closed = true;
      events.close();
      wake?.();
    };
    events.onmessage = (event) => {
      if (closed) return;
      try {
        const snapshot = ListSchema.parse(JSON.parse(event.data));
        if (snapshot.id !== listID) throw new Error("Unexpected list in stream.");
        // Full snapshots supersede each other if several arrive before consumption.
        pending = snapshot;
        wake?.();
      } catch (cause) {
        error = cause;
        close();
      }
    };
    events.onerror = () => {
      // EventSource reconnects transient failures itself.
      if (events.readyState !== EventSource.CLOSED) return;
      error = new Error("Unable to open the list stream.");
      close();
    };
    events.addEventListener("deleted", () => {
      if (closed) return;
      close();
      onDeleted();
    });
    signal.addEventListener("abort", close, { once: true });
    try {
      while (!closed) {
        if (pending) {
          const snapshot = pending;
          pending = undefined;
          yield snapshot;
        } else {
          await new Promise<void>((resolve) => {
            wake = resolve;
          });
          wake = undefined;
        }
      }
      if (error) throw error;
    } finally {
      signal.removeEventListener("abort", close);
      close();
    }
  },
});
