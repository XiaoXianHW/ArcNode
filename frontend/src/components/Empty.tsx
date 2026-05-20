export function Empty({ message = 'No data yet' }: { message?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-10 text-center">
      <div className="h-8 w-8 rounded-full border border-border" />
      <p className="mt-3 text-sm text-muted">{message}</p>
    </div>
  );
}

export function ErrorState({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : typeof error === 'string' ? error : JSON.stringify(error);
  return (
    <div className="rounded-md border border-border bg-elevated p-4 text-sm text-fg/80">
      <p className="text-xs uppercase tracking-wider text-muted mb-1">Error</p>
      <p className="font-mono text-fg">{message}</p>
    </div>
  );
}
