import { useEffect, useState } from "react";
import type { ReactNode } from "react";

export function navigate(path: string) {
  window.history.pushState({}, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

export function usePath() {
  const [path, setPath] = useState(window.location.pathname + window.location.search);

  useEffect(() => {
    const handler = () => setPath(window.location.pathname + window.location.search);
    window.addEventListener("popstate", handler);
    return () => window.removeEventListener("popstate", handler);
  }, []);

  return path;
}

export function Link(props: { href: string; children: ReactNode; className?: string }) {
  return (
    <a
      href={props.href}
      className={props.className}
      onClick={(event) => {
        event.preventDefault();
        navigate(props.href);
      }}
    >
      {props.children}
    </a>
  );
}
