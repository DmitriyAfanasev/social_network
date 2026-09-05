import { navigate } from "../../shared/lib/navigation";

/** Объясняет состояние административного интерфейса в Go-миграции. */
export function AdminPage() {
  return (
    <section className="panel edit-panel">
      <header className="page-header">
        <div><span className="eyebrow">Система</span><h1>Администрирование</h1></div>
        <button type="button" className="secondary" onClick={() => navigate("/")}>В ленту</button>
      </header>
      <div className="state-stack">
        <p>Admin и analytics в текущей архитектуре Go работают как фоновые Kafka-консьюмеры.</p>
        <p className="muted">HTTP-панель администратора ещё не входит в gateway-контракт, поэтому старые Python REST-запросы отключены.</p>
      </div>
    </section>
  );
}
