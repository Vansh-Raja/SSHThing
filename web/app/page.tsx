export default function HomePage() {
  return (
    <main style={{ maxWidth: 720, margin: "0 auto", padding: "64px 24px" }}>
      <h1>SSHThing Teams</h1>
      <p>Browser auth, team creation, invite acceptance, and account settings live here.</p>
      <ul>
        <li><a href="/login">Login</a></li>
        <li><a href="/signup">Sign up</a></li>
        <li><a href="/teams/create">Create team</a></li>
        <li><a href="/teams/join">Join team</a></li>
      </ul>
    </main>
  );
}

