import { Routes, Route, Link } from 'react-router-dom'
import HomePage from './pages/HomePage'
import PackagePage from './pages/PackagePage'
import SearchPage from './pages/SearchPage'
import AdoptionPage from './pages/AdoptionPage'
import UploadPage from './pages/UploadPage'
import ReviewsPage from './pages/ReviewsPage'

export default function App() {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <nav className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
          <Link to="/" className="text-lg font-semibold">
            Trove
          </Link>
          <div className="flex items-center gap-4">
            <Link to="/" className="text-sm text-muted-foreground hover:text-foreground">
              Packages
            </Link>
            <Link to="/search" className="text-sm text-muted-foreground hover:text-foreground">
              Search
            </Link>
            <Link to="/adoption" className="text-sm text-muted-foreground hover:text-foreground">
              Adoption
            </Link>
            <Link to="/reviews" className="text-sm text-muted-foreground hover:text-foreground">
              Reviews
            </Link>
            <Link to="/upload" className="text-sm text-primary hover:underline">
              Publish
            </Link>
          </div>
        </nav>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/adoption" element={<AdoptionPage />} />
          <Route path="/reviews" element={<ReviewsPage />} />
          <Route path="/upload" element={<UploadPage />} />
          <Route path="/packages/:org/:namespace/:name" element={<PackagePage />} />
        </Routes>
      </main>
    </div>
  )
}
