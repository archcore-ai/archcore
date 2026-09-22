---
title: "Server State via React Query (object API)"
status: accepted
tags: [frontend]
---

## Rule

All server state is fetched through React Query v5 using the object-style API.
Local UI state stays in component state or a store — never duplicate server data.

```ts
const { data, isPending } = useQuery({
  queryKey: ["projects", filters],
  queryFn: () => api.getProjects(filters),
  staleTime: 30_000,
});
```

## Rationale

One cache, one source of truth for remote data. The object API is the v5
standard; the v4 array-overload is deprecated.
