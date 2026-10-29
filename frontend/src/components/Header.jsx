import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth.jsx";

export default function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  return (
    <header className="site-header">
      <Link to="/" className="brand">Livepoll</Link>
      <nav>
        {user ? (
          <>
            <Link to="/dashboard">My polls</Link>
            <Link className="btn small" to="/create">New poll</Link>
            <button
              className="link-btn"
              onClick={() => {
                logout();
                navigate("/");
              }}
            >
              Log out
            </button>
          </>
        ) : (
          <>
            <Link to="/login">Log in</Link>
            <Link className="btn small" to="/signup">Sign up</Link>
          </>
        )}
      </nav>
    </header>
  );
}
