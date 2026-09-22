---
title: "Migrate a Fetch Hook to React Query"
status: accepted
tags: [frontend]
---

## Pattern change

Replace ad-hoc `useEffect` + `useState` fetching with a React Query `useQuery`.

## Before

```ts
const [data, setData] = useState(null);
useEffect(() => { api.get(id).then(setData); }, [id]);
```

## After

```ts
const { data } = useQuery({ queryKey: ["item", id], queryFn: () => api.get(id) });
```

## Why

Removes manual loading/error/caching code and dedupes in-flight requests.
