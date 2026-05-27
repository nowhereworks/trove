import { Routes, Route, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import HomePage from './pages/HomePage'
import PackagePage from './pages/PackagePage'
import SearchPage from './pages/SearchPage'
import AdoptionPage from './pages/AdoptionPage'
import UploadPage from './pages/UploadPage'
import ReviewsPage from './pages/ReviewsPage'
import CreateOrgPage from './pages/CreateOrgPage'
import LoginPage from './pages/LoginPage'
import { api } from './lib/api'
import { AuthProvider, useAuth } from './lib/auth'

export default function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  )
}

function AppContent() {
  const { data: config } = useQuery({ queryKey: ['config'], queryFn: api.getConfig })
  const { user, isAuthenticated, logout } = useAuth()
  const [showUserMenu, setShowUserMenu] = useState(false)

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
            {(config?.allowCreateOrg ?? true) && (
              <Link to="/orgs/new" className="text-sm text-muted-foreground hover:text-foreground">
                New Org
              </Link>
            )}
            {isAuthenticated ? (
              <div className="relative">
                <button
                  onClick={() => setShowUserMenu(!showUserMenu)}
                  className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
                >
                  <span className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center text-xs font-medium text-primary">
                    {user?.displayName?.charAt(0).toUpperCase() ?? 'U'}
                  </span>
                </button>
                {showUserMenu && (
                  <div className="absolute right-0 mt-2 w-48 bg-background border rounded-lg shadow-lg py-1 z-50">
                    <div className="px-3 py-2 border-b">
                      <p className="text-sm font-medium">{user?.displayName}</p>
                      <p className="text-xs text-muted-foreground">{user?.email}</p>
                    </div>
                    <button
                      onClick={() => { setShowUserMenu(false); logout(); }}
                      className="w-full text-left px-3 py-2 text-sm text-red-600 hover:bg-muted"
                    >
                      Sign out
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <Link to="/login" className="text-sm text-primary hover:underline">
                Sign in
              </Link>
            )}
          </div>
        </nav>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/adoption" element={<AdoptionPage />} />
          <Route path="/reviews" element={<ReviewsPage />} />
          <Route path="/orgs/new" element={<CreateOrgPage />} />
          <Route path="/upload" element={<UploadPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/packages/:org/:namespace/:name" element={<PackagePage />} />
        </Routes>
      </main>
    </div>
  )
}
