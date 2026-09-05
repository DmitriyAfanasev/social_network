import { navigate } from "../../shared/lib/navigation";

type PlaceholderPageProps = {
  title: string;
  description: string;
};

export function PlaceholderPage({ title, description }: PlaceholderPageProps) {
  return (
    <section className="placeholder-page">
      <div className="placeholder-hero">
        <span className="eyebrow">Скоро</span>
        <h1>{title}</h1>
        <p>{description}</p>
        <div className="placeholder-actions">
          <button onClick={() => navigate("/")}>Открыть ленту</button>
          <button className="secondary" onClick={() => navigate("/settings")}>
            Настройки
          </button>
        </div>
      </div>
      <div className="placeholder-list">
        <div />
        <div />
        <div />
      </div>
    </section>
  );
}
