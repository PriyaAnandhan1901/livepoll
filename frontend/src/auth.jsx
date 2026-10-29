import { createContext, useCallback, useContext, useState } from "react";
import { request } from "./api.js";

const Ctx = createContext(null);

function load() {
  try {
    const user = JSON.parse(localStorage.getItem("lp_user"));
    return user && localStorage.getItem("lp_token") ? user : null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(load);

  const save = (data) => {
    try {
      localStorage.setItem("lp_token", data.token);
      localStorage.setItem("lp_user", JSON.stringify(data.user));
    } catch {
      /* ignore */
    }
    setUser(data.user);
  };

  const login = async (email, password) =>
    save(await request("/auth/login", { method: "POST", body: { email, password } }));
  const signup = async (name, email, password) =>
    save(await request("/auth/signup", { method: "POST", body: { name, email, password } }));
  const logout = useCallback(() => {
    try {
      localStorage.removeItem("lp_token");
      localStorage.removeItem("lp_user");
    } catch {
      /* ignore */
    }
    setUser(null);
  }, []);

  return <Ctx.Provider value={{ user, login, signup, logout }}>{children}</Ctx.Provider>;
}

export const useAuth = () => useContext(Ctx);
