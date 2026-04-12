import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="grid min-h-screen place-items-center px-5">
      <div className="text-center">
        <h1 className="font-display text-3xl font-semibold">404</h1>
        <p className="mt-2 text-slate-500">页面不存在。</p>
        <Link to="/" className="mt-4 inline-block rounded-xl bg-ink px-4 py-2 text-sm text-white">
          返回首页
        </Link>
      </div>
    </div>
  );
}
