export function FeedGridSkeleton({ count = 4 }: { count?: number }) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {Array.from({ length: count }).map((_, index) => (
        <div key={index} className="h-[320px] rounded-[28px] bg-white shadow-card" />
      ))}
    </div>
  );
}
