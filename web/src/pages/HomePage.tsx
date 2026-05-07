/**
 * HomePage is intentionally small for the first TDD slice.
 * We'll expand it as feature stories are added.
 */
export function HomePage() {
  return (
    <main aria-labelledby="home-title" className="home">
      <h1 id="home-title">NextWork</h1>
      <p className="home-tagline">Browse classes and sign up when enrollment opens.</p>
    </main>
  )
}
