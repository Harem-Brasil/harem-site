import { forumRooms, trendingTopics } from "../mockData";

export default function RightPanel() {
  return (
    <aside className="right-panel">
      <section className="card">
        <h3>Salas do fórum</h3>
        <ul className="forum-list">
          {forumRooms.map((room) => (
            <li key={room}>
              <span>#</span>
              {room}
            </li>
          ))}
        </ul>
      </section>

      <section className="card">
        <h3>Tópicos em alta</h3>
        {trendingTopics.map((topic) => (
          <div className="topic" key={topic.title}>
            <p className="topic-title">{topic.title}</p>
            <small>{topic.replies} respostas</small>
          </div>
        ))}
      </section>
    </aside>
  );
}
