import { navigate } from "../../shared/lib/navigation";

export function NotFoundPage() {
  return (
    <section className="empty-state">
      <h1>Страница не найдена</h1>
      <button onClick={() => navigate("/")}>Вернуться в ленту</button>
    </section>
  );
}
