import { navigate } from "../../router";
import { get } from "svelte/store";
import { queryClient } from "../query";
import { session } from "../user/session";
import type { ListSummary } from "./api";
import { summariesKey } from "./queries";

export const navigateToDefaultList = async () => {
  const lists = queryClient.getQueryData<ListSummary[]>(summariesKey(get(session)?.id)) ?? [];
  const defaultList = lists.at(0);

  if (lists.length === 1 && defaultList) {
    return navigate("/lists/:id", {
      params: { id: defaultList.id },
    });
  } else {
    return navigate("/lists", { replace: true });
  }
};
