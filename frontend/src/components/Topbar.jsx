export default function Topbar() {
  return (
    <header className="topbar">
      <input
        type="search"
        className="search"
        placeholder="Procurar criadoras, tópicos e posts"
      />
      <div className="top-actions">
        <button className="ghost-btn">Notificações</button>
        <button className="ghost-btn">Config</button>
      </div>
    </header>
  );
}
