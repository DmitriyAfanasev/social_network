import type { ReactNode } from "react";

type AuthLayoutProps = {
  title: string;
  aside: ReactNode;
  children: ReactNode;
};

export function AuthLayout({ title, aside, children }: AuthLayoutProps) {
  return (
    <section className="auth-layout">
      <div className="auth-copy">
        <span className="eyebrow">Social API</span>
        <h1>{title}</h1>
        <p>{aside}</p>
      </div>
      <div className="panel auth-panel">{children}</div>
    </section>
  );
}
