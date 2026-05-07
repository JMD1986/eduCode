import { NavLink, Outlet } from 'react-router-dom'

function navClassName({ isActive }: { isActive: boolean }): string {
  return isActive ? 'nav-link nav-link-active' : 'nav-link'
}

/**
 * Shell shared by every screen: primary nav + routed content via Outlet.
 * NavLink sets aria-current="page" on the active route for accessibility.
 */
export function AppLayout() {
  return (
    <>
      <header className="app-header">
        <div className="app-header-inner">
          <NavLink className="app-brand" to="/" end>
            NextWork
          </NavLink>
          <nav aria-label="Primary navigation">
            <ul className="nav-list">
              <li>
                <NavLink className={navClassName} to="/" end>
                  Home
                </NavLink>
              </li>
              <li>
                <NavLink className={navClassName} to="/classes">
                  Classes
                </NavLink>
              </li>
              <li>
                <NavLink className={navClassName} to="/my-classes">
                  My classes
                </NavLink>
              </li>
            </ul>
          </nav>
        </div>
      </header>
      <Outlet />
    </>
  )
}
