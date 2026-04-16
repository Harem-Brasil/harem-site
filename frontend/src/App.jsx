import { useState } from "react";
import { Outlet } from "react-router-dom";
import { initialPosts } from "./mockData";
import RightPanel from "./components/RightPanel";
import Sidebar from "./components/Sidebar";
import Topbar from "./components/Topbar";

export default function App() {
  const [posts, setPosts] = useState(initialPosts);

  function publishPost(text, pageType) {
    const value = text.trim();
    if (!value) return;

    setPosts((prev) => [
      {
        id: crypto.randomUUID(),
        author: "Você",
        time: "agora",
        text: value,
        tags: pageType === "forum" ? ["forum"] : ["feed"],
      },
      ...prev,
    ]);
  }

  return (
    <div className="layout">
      <Sidebar />

      <main className="content">
        <Topbar />
        <Outlet context={{ posts, publishPost }} />
      </main>

      <RightPanel />
    </div>
  );
}
