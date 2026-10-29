import { Link, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { useAuth } from "./auth.jsx";
import Header from "./components/Header.jsx";
import Home from "./pages/Home.jsx";
import Auth from "./pages/Auth.jsx";
import Dashboard from "./pages/Dashboard.jsx";
import Create from "./pages/Create.jsx";
import Poll from "./pages/Poll.jsx";

function RequireAuth({ children }) {
  const { user } = useAuth();
  const location = useLocation();
  if (!user) return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  return children;
}

function NotFound() {
  return (
    <main className="page narrow">
      <h1>Page not found</h1>
      <p className="muted">That link doesn't lead anywhere. Check the address, or start from the home page.</p>
      <Link className="btn" to="/">Go to home</Link>
    </main>
  );
}

export default function App() {
  return (
    <>
      <Header />
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Auth mode="login" />} />
        <Route path="/signup" element={<Auth mode="signup" />} />
        <Route path="/dashboard" element={<RequireAuth><Dashboard /></RequireAuth>} />
        <Route path="/create" element={<RequireAuth><Create /></RequireAuth>} />
        <Route path="/p/:id" element={<Poll />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </>
  );
}
