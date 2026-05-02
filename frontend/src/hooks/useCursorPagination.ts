import { useState, useEffect, useCallback } from 'react';
import { reportError } from '../lib/errorReporter';

interface UseCursorPaginationProps<T, Req, Resp> {
  clientMethod: (req: Req) => Promise<Resp>;
  requestBuilder: (cursor: string) => Req;
  responseExtractor: (res: Resp) => { items: T[], nextCursor: string };
  storeSetter: (items: T[]) => void;
  initialItems: T[];
}

export function useCursorPagination<T, Req, Resp>({
  clientMethod,
  requestBuilder,
  responseExtractor,
  storeSetter,
  initialItems,
}: UseCursorPaginationProps<T, Req, Resp>) {
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<T[]>(initialItems);

  const loadMore = useCallback(async () => {
    if (loading || !hasMore) return;

    setLoading(true);
    try {
      const request = requestBuilder(cursor);
      const response = await clientMethod(request);
      const { items: newItems, nextCursor } = responseExtractor(response);
      
      const updatedItems = [...items, ...newItems];
      setItems(updatedItems);
      storeSetter(updatedItems);
      setCursor(nextCursor);
      setHasMore(!!nextCursor);
    } catch (error) {
      reportError('useCursorPagination', error);
    } finally {
      setLoading(false);
    }
  }, [cursor, loading, hasMore, clientMethod, requestBuilder, responseExtractor, items, storeSetter]);

  useEffect(() => {
    setItems(initialItems);
  }, [initialItems]);

  return {
    items,
    hasMore,
    loadMore,
    loading,
  };
}
