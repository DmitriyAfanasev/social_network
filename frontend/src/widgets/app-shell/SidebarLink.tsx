import { Link } from "../../shared/lib/navigation";

type SidebarLinkProps = {
  href: string;
  label: string;
  path: string;
  badge?: string;
};

export function SidebarLink({ href, label, path, badge }: SidebarLinkProps) {
  const isActive =
    path === href ||
    (href !== "/" && path.startsWith(href)) ||
    (label === "Профиль" && path.startsWith("/profile/"));

  return (
    <Link href={href} className={isActive ? "nav-link active" : "nav-link"}>
      <span className="nav-dot" />
      <span>{label}</span>
      {badge && <span className="nav-badge">{badge}</span>}
    </Link>
  );
}
