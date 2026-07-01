/**
 * Format an ISO 8601 date string to a readable Chinese locale string.
 * Returns '-' for null, undefined, empty, or Go zero-value dates.
 */
export function formatDate(date: string | null | undefined): string {
  if (!date || date === '0001-01-01T00:00:00Z' || date.startsWith('0001-01-01')) {
    return '-';
  }
  try {
    return new Date(date).toLocaleString('zh-CN');
  } catch {
    return date;
  }
}
