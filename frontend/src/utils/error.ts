export function extractErrorMessage(e: any, fallback: string): string {
  return e?.response?.data?.msg || e?.message || fallback
}
