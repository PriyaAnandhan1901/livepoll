# Livepoll

A live polling tool. Create a poll, share the link, and everyone watching sees results update in real time with no refresh.

Live app: https://livepoll-app-smoky.vercel.app
API health check: https://livepoll-cpgb.onrender.com/healthz

Stack: React, Go (Gin), MongoDB, Redis

## Flow
Sign up, create a poll, share the link, audience votes (no login needed), results update live.

## Run locally
Backend:
1. cd backend
2. copy .env.example to .env and fill in MONGO_URI, REDIS_URL, JWT_SECRET
3. go run .

Frontend:
1. cd frontend
2. copy .env.example to .env (VITE_API_URL points at the backend)
3. npm install, then npm run dev

## Key decisions
- Redis drives the live part. Vote counts live in a hash per poll. A Lua script checks for a duplicate vote and increments in one atomic step, so two quick votes can never double count. Pub/sub pushes each change to every open viewer.
- MongoDB stores users and polls, plus a durable copy of the counts. If Redis loses its data, counts are rebuilt from MongoDB.
- Live updates use Server-Sent Events. Updates only flow server to client, and SSE works on free hosting without extra setup.
- Auth is JWT with bcrypt-hashed passwords. Only logged-in users can create, close or delete polls, and only the owner can manage a poll. Voting needs no login.
- All input is validated on the backend: lengths, option count, duplicates, ids. Request bodies are size-limited, and auth and vote routes are rate limited per IP in Redis.
- Frontend and backend live in separate folders and deploy separately (Vercel and Render).

## Known limits
- One vote per browser per poll, using an anonymous voter id. Blocking by IP would stop a whole classroom or office on shared wifi from voting. Clearing browser storage allows another vote.
- The backend runs on a free plan, so the first request after idle time can take up to a minute.
