// Helpers shared by the pages.

// errorText turns a rejected service call into a message for the page.
export function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
// formatDateTime formats an ISO timestamp (e.g. when a note was created) in local time.
export const formatDateTime = (iso: string) => dateTime.format(new Date(iso));
