export function UserCardSkeletonGrid({ count = 4 }: { count?: number }) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="rounded-[28px] border border-slate-200 bg-white p-5 shadow-card"
        >
          <div className="flex items-start gap-4">
            <div className="h-16 w-16 rounded-full bg-slate-100" />
            <div className="min-w-0 flex-1 space-y-3">
              <div className="h-6 w-28 rounded-full bg-slate-100" />
              <div className="h-4 w-full rounded-full bg-slate-100" />
              <div className="h-4 w-5/6 rounded-full bg-slate-100" />
              <div className="flex gap-3 pt-2">
                <div className="h-9 w-24 rounded-full bg-slate-100" />
                <div className="h-9 w-20 rounded-full bg-slate-100" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
