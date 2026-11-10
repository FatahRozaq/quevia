export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6 text-center">
      <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-400 mb-6">
        <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
        Sistem Operasional Aktif
      </div>

      <h1 className="text-4xl font-extrabold tracking-tight sm:text-6xl max-w-3xl bg-gradient-to-b from-white to-zinc-400 bg-clip-text text-transparent">
        Quevia Smart Queue &amp; Appointment
      </h1>

      <p className="mt-4 text-base sm:text-lg text-zinc-400 max-w-xl">
        Platform SaaS Multi-Tenant terpadu untuk penjadwalan mandiri anti-bentrok dan antrean real-time berstandar enterprise.
      </p>

      <div className="mt-8 flex flex-wrap items-center justify-center gap-4">
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4 text-left w-64 backdrop-blur">
          <p className="text-xs font-semibold uppercase tracking-wider text-zinc-500">Backend API</p>
          <p className="mt-1 text-sm font-medium text-zinc-200">Go Hertz + River Worker</p>
          <p className="mt-2 text-xs text-emerald-400 font-mono">Port 8080 • /healthz OK</p>
        </div>

        <div className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4 text-left w-64 backdrop-blur">
          <p className="text-xs font-semibold uppercase tracking-wider text-zinc-500">Frontend Web</p>
          <p className="mt-1 text-sm font-medium text-zinc-200">Next.js 14 App Router</p>
          <p className="mt-2 text-xs text-emerald-400 font-mono">Port 3000 • Responsive</p>
        </div>
      </div>
    </main>
  );
}
