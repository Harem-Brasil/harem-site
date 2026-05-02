import { NavLink } from "react-router-dom";

const items = [
  { label: "Início", to: "/home" },
  { label: "Fórum", to: "/forum" },
  { label: "Mensagens", to: "/mensagens" },
  { label: "Assinaturas", to: "/assinaturas" },
  { label: "Perfil", to: "/perfil" },
];

export default function Sidebar() {
  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-dot" />
        <h1>Harém Brasil</h1>
      </div>

      <nav className="menu">
        {items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              `menu-item menu-link ${isActive ? "active" : ""}`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>

      <button className="primary-btn">Nova publicação</button>
    </aside>
  );
}
