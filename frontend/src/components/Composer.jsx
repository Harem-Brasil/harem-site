import { useState } from "react";

export default function Composer({ title, pageType, onPublish }) {
  const [newPost, setNewPost] = useState("");

  function handlePublish() {
    onPublish(newPost, pageType);
    setNewPost("");
  }

  return (
    <section className="composer card">
      <h2>{title}</h2>
      <textarea
        value={newPost}
        onChange={(event) => setNewPost(event.target.value)}
        placeholder="Compartilhe uma atualização discreta..."
      />
      <div className="composer-actions">
        <div className="chips">
          <span className="chip">Foto</span>
          <span className="chip">Vídeo</span>
          <span className="chip">Enquete</span>
        </div>
        <button className="primary-btn small" onClick={handlePublish}>
          Publicar
        </button>
      </div>
    </section>
  );
}
