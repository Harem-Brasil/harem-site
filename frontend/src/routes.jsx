import { Navigate, Route, Routes } from "react-router-dom";
import App from "./App";
import ForumPage from "./pages/ForumPage";
import HomePage from "./pages/HomePage";
import PlaceholderPage from "./pages/PlaceholderPage";

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<App />}>
        <Route index element={<Navigate to="/home" replace />} />
        <Route path="home" element={<HomePage />} />
        <Route path="forum" element={<ForumPage />} />
        <Route path="mensagens" element={<PlaceholderPage title="Mensagens" />} />
        <Route
          path="assinaturas"
          element={<PlaceholderPage title="Assinaturas" />}
        />
        <Route path="perfil" element={<PlaceholderPage title="Perfil" />} />
      </Route>
    </Routes>
  );
}
